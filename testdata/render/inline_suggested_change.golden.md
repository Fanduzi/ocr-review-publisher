**OCR Review Publisher**

Error return value of `fmt.Println` is not checked.

📍 `service/user.go:37`

Suggested change:

```go
if err != nil {
	return fmt.Errorf("log error: %w", err)
}
```

<!-- ocr-review-publisher:inline -->
