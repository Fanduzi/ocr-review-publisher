package main

import "os"

// resolveString returns the first non-empty value from flag, then env vars, then default.
func resolveString(flagVal string, envKeys []string, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return defaultVal
}

// resolveInt returns flag value if set (>0), then parses env var, then default.
func resolveInt(flagVal int, envKeys []string, defaultVal int) int {
	if flagVal > 0 {
		return flagVal
	}
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			n := 0
			for _, c := range v {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				} else {
					break
				}
			}
			if n > 0 {
				return n
			}
		}
	}
	return defaultVal
}

// gitlabConfig resolves GitLab connection parameters from flags and env.
type gitlabConfig struct {
	BaseURL string
	Token   string
	Project string
	MRIID   int
}

func resolveGitLabConfig(baseURL, token, project string, mrIID int) gitlabConfig {
	return gitlabConfig{
		BaseURL: resolveString(baseURL, []string{"OCR_GITLAB_BASE_URL", "CI_SERVER_URL"}, "https://gitlab.com"),
		Token:   resolveString(token, []string{"GITLAB_TOKEN", "OCR_GITLAB_TOKEN"}, ""),
		Project: resolveString(project, []string{"CI_PROJECT_ID"}, ""),
		MRIID:   resolveInt(mrIID, []string{"CI_MERGE_REQUEST_IID"}, 0),
	}
}
