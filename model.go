package componentizer

type (
	TemplateContext interface {
		Clone(ref ComponentRef) TemplateContext
		Execute(content string) (string, error)
	}
)
