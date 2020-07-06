package componentizer

type (
	Model interface {
		Merge(with Model) (Model, error)
	}

	Component interface {
		ComponentRef
		Repository() Repository
		Descriptor() string
		Templated() (bool, []string)
		ParseModel(path string, tplC TemplateContext) (Model, error)
		ParseRefs(path string, tplC TemplateContext) ([]ComponentRef, Component, error)
	}

	// ComponentRef allows to access to a component through its reference
	ComponentRef interface {
		// ComponentId returns the referenced component id
		ComponentId() string
		// HasComponent returns true if the reference is not empty
		HasComponent() bool
		// Component returns the referenced component
		Component(model interface{}) (Component, error)
	}
)
