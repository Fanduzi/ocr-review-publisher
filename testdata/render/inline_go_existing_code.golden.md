**OCR Review Publisher**

Error return value of `fmt.Println` is not checked.

📍 `service/user.go:37`

<details><summary>Review context</summary>

Existing code:

```go
fmt.Println(err)
```

Reviewer notes:

The error from fmt.Println is silently discarded.

</details>

<!-- ocr-review-publisher:inline -->
