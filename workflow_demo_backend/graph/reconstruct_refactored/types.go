package reconstruct_refactored

// RewriteState 用于在图处理过程中存储状态
type RewriteState struct {
	OriginalQuery  string
	Decision       string
	RewrittenQuery string
	Intent         string
}

