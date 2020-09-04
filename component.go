package componentizer

type (
	Model interface {
		IsReferenced(c Component) bool
		Merge(with Model) (Model, error)
	}

	Component interface {
		ComponentRef
		GetRepository() Repository
		GetTemplates() (bool, []string)
		ParseModel(path string, tplC TemplateContext) (Model, error)
		ParseComponents(path string, tplC TemplateContext) (Component, []Component, error)
	}

	TemplateContext interface {
		Clone(ref ComponentRef) TemplateContext
		Execute(content string) (string, error)
	}

	// ComponentRef allows to access to a component through its reference
	ComponentRef interface {
		// ComponentId returns the referenced component id
		ComponentId() string
		// Component returns the referenced component
		Component(model interface{}) (Component, error)
	}
)
