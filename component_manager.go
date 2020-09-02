package componentizer

import (
	"fmt"
	"log"
	"os"
)

type (
	//ComponentManager is the facade for accessing components.
	ComponentManager interface {
		//Init initialize the component manager with the specified main component
		Init(main Component, tplC TemplateContext) (Model, error)

		//ContainsFile returns paths pointing on templated components containing the given file
		//	Parameters
		//		name: the name of the file to search
		//		ctx: the context used to eventually template the matching components
		//		in: the component referencer where to look for the file, if not provided
		//          the Finder will look into all the components available into the platform.
		ContainsFile(name string, tplC TemplateContext, in ...ComponentRef) MatchingPaths

		//ContainsDirectory returns paths pointing on templated components containing the given directory
		//	Parameters
		//		name: the name of the directory to search
		//		ctx: the context used to eventually template the matching components
		//		in: the component referencer where to look for the directory, if not provided
		//          the Finder will look into all the components available into the platform.
		ContainsDirectory(name string, tplC TemplateContext, in ...ComponentRef) MatchingPaths

		//IsAvailable checks if a component is locally available
		IsAvailable(cr ComponentRef) bool

		// ComponentOrder returns a slice of component identifiers in the parsing order
		ComponentOrder() []string

		//Use returns a component matching the given reference.
		//If the component corresponding to the reference contains a template
		//definition then the component will be duplicated and templated before
		// being returned as a UsableComponent.
		// Don't forget to Release the UsableComponent once is processing is over...
		Use(cr ComponentRef, tplC TemplateContext) (UsableComponent, error)
	}

	componentManager struct {
		l         *log.Logger
		directory string
		fComps    map[string]fetchedComponent
		order     []string
	}

	fetchedComponent struct {
		id        string
		rootPath  string
		component Component
	}
)

//createComponentManager creates a new component manager
func CreateComponentManager(l *log.Logger, workDir string) ComponentManager {
	return &componentManager{
		l:         l,
		directory: workDir,
		fComps:    map[string]fetchedComponent{},
		order:     []string{},
	}
}

func (cm *componentManager) Init(main Component, tplC TemplateContext) (Model, error) {
	// Compute a temporary model with only the parents to find components
	tempModel, comps, err := cm.findComponents(main, tplC)
	if err != nil {
		return nil, err
	}

	// Go through found components to build the final model in order
	var fModel Model
	for _, comp := range comps {
		// Check if the component is referenced from the model
		if tempModel.IsReferenced(comp) {
			// Refresh component by resolving it again (takes into account overrides after first discovery)
			comp, err := comp.Component(tempModel)
			if err != nil {
				return nil, err
			}

			// Fetch the component if necessary
			fComp, err := cm.fetchComponent(comp)
			if err != nil {
				return nil, err
			}

			// Parse the component model to merge it into the final model
			cModel, err := comp.ParseModel(fComp.rootPath, tplC)
			if err != nil {
				return nil, err
			}
			if cModel != nil {
				if fModel != nil {
					fModel, err = fModel.Merge(cModel)
					if err != nil {
						return nil, err
					}
				} else {
					fModel = cModel
				}
			}

			// Add the component id to the ordered list
			cm.order = append(cm.order, comp.ComponentId())
		}
	}

	// Update fetched components with refreshed components from the model
	for fId, fComp := range cm.fComps {
		fComp.component, err = fComp.component.Component(fModel)
		if err != nil {
			return nil, err
		}
		cm.fComps[fId] = fComp
	}

	return fModel, nil
}

func (cm componentManager) findComponents(comp Component, tplC TemplateContext) (Model, []Component, error) {
	var fModel Model
	var comps []Component

	// Fetch component
	fComp, err := cm.fetchComponent(comp)
	if err != nil {
		cm.l.Printf("error fetching the main descriptor %s", err.Error())
		return nil, nil, err
	}

	// Parse references
	parent, otherComps, err := comp.ParseComponents(fComp.rootPath, tplC)
	if err != nil {
		return nil, nil, err
	}

	// Go through parents recursively
	if parent != nil {
		var pComps []Component
		fModel, pComps, err = cm.findComponents(parent, tplC)
		if err != nil {
			return nil, nil, err
		}
		comps = append(comps, pComps...)
	}

	// Add the discovered components to the list
	comps = append(comps, otherComps...)
	comps = append(comps, comp)

	// Parse the component model
	cModel, err := comp.ParseModel(fComp.rootPath, tplC)
	if err != nil {
		return nil, nil, err
	}
	if cModel != nil {
		if fModel != nil {
			fModel, err = fModel.Merge(cModel)
			if err != nil {
				return nil, nil, err
			}
		} else {
			fModel = cModel
		}
	}

	return fModel, comps, nil
}

func (cm componentManager) ContainsFile(name string, tplC TemplateContext, in ...ComponentRef) MatchingPaths {
	return cm.contains(false, name, tplC, in...)
}

func (cm componentManager) ContainsDirectory(name string, tplC TemplateContext, in ...ComponentRef) MatchingPaths {
	return cm.contains(true, name, tplC, in...)
}

func (cm componentManager) IsAvailable(cr ComponentRef) bool {
	_, ok := cm.fComps[cr.ComponentId()]
	return ok
}

func (cm componentManager) ComponentOrder() []string {
	return cm.order
}

func (cm componentManager) Use(cr ComponentRef, tplC TemplateContext) (UsableComponent, error) {
	var res usable
	fetchedC, ok := cm.fComps[cr.ComponentId()]
	if !ok {
		return nil, fmt.Errorf("component %s is not available", cr.ComponentId())
	}
	if ok, patterns := fetchedC.component.Templated(); ok {
		templatedPath, err := executeTemplate(fetchedC.rootPath, patterns, tplC.Clone(cr))
		if err != nil {
			return usable{}, err
		}

		if templatedPath != "" {
			// Path has a value, the component has been templated
			res = usable{
				id:        cr.ComponentId(),
				path:      templatedPath,
				release:   cm.cleanup(templatedPath),
				templated: true,
			}
		} else {
			res = usable{
				id:        cr.ComponentId(),
				path:      fetchedC.rootPath,
				templated: false,
			}
		}
	} else {
		res = usable{
			id:        cr.ComponentId(),
			path:      fetchedC.rootPath,
			templated: false,
		}
	}

	// Keep component env vars TODO fix this
	//if o, ok := cr.(EnvVarsAware); ok {
	//	res.envVars = o.EnvVars()
	//} else {
	//	res.envVars = model.EnvVars{}
	//}

	return res, nil
}

func (cm componentManager) contains(isFolder bool, name string, tplC TemplateContext, in ...ComponentRef) MatchingPaths {
	res := MatchingPaths{
		Paths: make([]MatchingPath, 0, 0),
	}
	if len(in) > 0 {
		for _, cRef := range in {
			if match, b := cm.checkMatch(cRef, tplC, name, isFolder); b {
				res.Paths = append(res.Paths, match)
			}
		}
	} else {
		for _, comp := range cm.fComps {
			if match, b := cm.checkMatch(comp.component, tplC, name, isFolder); b {
				res.Paths = append(res.Paths, match)
			}
		}
	}
	return res
}

func (cm *componentManager) isComponentFetched(id string) (val fetchedComponent, present bool) {
	val, present = cm.fComps[id]
	return
}

func (cm *componentManager) fetchComponent(c Component) (fetchedComponent, error) {
	fComp, isFetched := cm.isComponentFetched(c.ComponentId())
	if !isFetched {
		cm.l.Printf("Fetching component %s", c.ComponentId())

		// Resolve fetch handler
		h, err := GetScmHandler(cm.l, cm.directory, c)
		if err != nil {
			return fetchedComponent{}, err
		}

		// Do the fetching
		fComp, err = h()
		if err != nil {
			cm.l.Printf("error fetching the component: %s", err.Error())
			return fetchedComponent{}, err
		}

		// Register the component
		fComp.component = c
		cm.fComps[c.ComponentId()] = fComp
		cm.l.Printf("Component %s is available in %s", c.ComponentId(), fComp.rootPath)
	}
	return fComp, nil
}

func (cm componentManager) checkMatch(r ComponentRef, tplC TemplateContext, name string, isFolder bool) (MatchingPath, bool) {
	uv, err := cm.Use(r, tplC)
	if err != nil {
		cm.l.Printf("An error occurred using the component %s : %s", r.ComponentId(), err.Error())
		return mPath{}, false
	}
	if isFolder {
		if ok, match := uv.ContainsDirectory(name); ok {
			return match, true
		} else {
			uv.Release()
		}
	} else {
		if ok, match := uv.ContainsFile(name); ok {
			return match, true
		} else {
			uv.Release()
		}
	}
	return mPath{}, false
}

func (cm componentManager) cleanup(path string) func() {
	return func() {
		err := os.RemoveAll(path)
		if err != nil {
			cm.l.Printf("Unable to clean temporary component path %s: %s", path, err.Error())
		}
	}
}
