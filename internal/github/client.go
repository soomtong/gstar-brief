package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL      = "https://api.github.com"
	maxFileSize  = 10 * 1024 // 10KB
	maxCodeFiles = 3
	perPage      = 100
)

// Client는 GitHub REST API 클라이언트입니다.
type Client struct {
	token      string
	httpClient *http.Client
}

// Repo는 starred 저장소의 메타데이터입니다.
type Repo struct {
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	Language        *string   `json:"language"`
	StargazersCount int       `json:"stargazers_count"`
	Topics          []string  `json:"topics"`
	HTMLURL         string    `json:"html_url"`
	StarredAt       time.Time `json:"-"`
}

type starredItem struct {
	StarredAt time.Time `json:"starred_at"`
	Repo      Repo      `json:"repo"`
}

type contentResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Name     string `json:"name"`
	Size     int    `json:"size"`
	Type     string `json:"type"`
	Path     string `json:"path"`
}

// New는 새 GitHub 클라이언트를 생성합니다.
func New(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request 생성 실패: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API Rate Limit 초과 (remaining=%s, reset=%s)", remaining, reset)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API 오류 %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// starredPages는 페이지 단위로 starred 아이템을 yield하는 이터레이터입니다.
// 각 페이지의 아이템을 순서대로 yield하며, 에러 발생 시 중단됩니다.
func (c *Client) starredPages(ctx context.Context, username string) iter.Seq2[starredItem, error] {
	return func(yield func(starredItem, error) bool) {
		for page := 1; ; page++ {
			url := fmt.Sprintf("%s/users/%s/starred?per_page=%d&page=%d", baseURL, username, perPage, page)
			resp, err := c.do(ctx, "GET", url, map[string]string{
				"Accept": "application/vnd.github.star+json",
			})
			if err != nil {
				yield(starredItem{}, err)
				return
			}

			var items []starredItem
			if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
				resp.Body.Close()
				yield(starredItem{}, fmt.Errorf("응답 파싱 실패: %w", err))
				return
			}
			resp.Body.Close()

			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}

			// 마지막 페이지면 종료
			if len(items) < perPage {
				return
			}
		}
	}
}

// ListStarred는 유저의 스타 저장소 목록을 반환합니다.
// offset: 건너뛸 아이템 수 (최신 순 기준, 0이면 처음부터)
// limit:  수집할 최대 아이템 수 (0이면 무제한)
func (c *Client) ListStarred(ctx context.Context, username string, offset, limit int) ([]Repo, error) {
	var repos []Repo
	var skipped int

	for item, err := range c.starredPages(ctx, username) {
		if err != nil {
			return nil, err
		}

		if skipped < offset {
			skipped++
			continue
		}

		r := item.Repo
		r.StarredAt = item.StarredAt
		repos = append(repos, r)

		if limit > 0 && len(repos) >= limit {
			break
		}
	}

	return repos, nil
}

// GetReadme는 저장소의 README 내용을 반환합니다.
func (c *Client) GetReadme(ctx context.Context, fullName string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/readme", baseURL, fullName)
	resp, err := c.do(ctx, "GET", url, nil)
	if err != nil {
		// README 없는 저장소는 빈 문자열 반환
		return "", nil
	}
	defer resp.Body.Close()

	var content contentResponse
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", fmt.Errorf("README 파싱 실패: %w", err)
	}

	if content.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
		if err != nil {
			return "", fmt.Errorf("base64 디코딩 실패: %w", err)
		}
		return string(decoded), nil
	}

	return content.Content, nil
}

// priorityFiles는 코드 분석 시 우선 탐색할 파일 이름 목록입니다.
var priorityFiles = []string{
	"main.go", "cmd/main.go",
	"index.ts", "src/index.ts",
	"index.js", "src/index.js",
	"app.py", "main.py",
	"lib.rs", "src/main.rs",
	"index.rb", "app.rb",
}

// GetCodeSnippets는 저장소의 주요 코드 파일 내용을 반환합니다.
func (c *Client) GetCodeSnippets(ctx context.Context, fullName string) ([]string, error) {
	var snippets []string

	for _, path := range priorityFiles {
		if len(snippets) >= maxCodeFiles {
			break
		}

		url := fmt.Sprintf("%s/repos/%s/contents/%s", baseURL, fullName, path)
		resp, err := c.do(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		var content contentResponse
		if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if content.Type != "file" || content.Size > maxFileSize {
			continue
		}

		if content.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
			if err != nil {
				continue
			}
			snippets = append(snippets, fmt.Sprintf("// %s\n%s", path, string(decoded)))
		}
	}

	return snippets, nil
}

// RateLimitRemaining은 현재 Rate Limit 잔여 횟수를 반환합니다.
func (c *Client) RateLimitRemaining(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/rate_limit", baseURL)
	resp, err := c.do(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	remaining := resp.Header.Get("X-RateLimit-Remaining")
	n, err := strconv.Atoi(remaining)
	if err != nil {
		return 0, nil
	}
	return n, nil
}
