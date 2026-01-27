module example

go 1.21

replace github.com/langfuse/langfuse-go => ../../

require (
	github.com/langfuse/langfuse-go v0.0.0
	github.com/sashabaranov/go-openai v1.20.4
)

require github.com/google/uuid v1.6.0
