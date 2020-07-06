package componentizer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	}

	fetchedComponent struct {
		id        string
		rootPath  string
		component Component
	}
)

//createComponentManager creates a new component manager
func CreateComponentManager(l *log.Logger, workDir string) ComponentManager {
	return componentManager{
		l:         l,
		directory: workDir,
		fComps:    map[string]fetchedComponent{},
	}
}

func (cm componentManager) Init(main Component, tplC TemplateContext) (Model, error) {
	var fM Model

	// Fetch component
	fComp, err := cm.fetchComponent(main)
	if err != nil {
		cm.l.Printf("error fetching the main descriptor %s", err.Error())
		return nil, err
	}

	// Parse all its references
	var refs []ComponentRef
	var parent Component
	var hasDescriptor bool
	descPath := filepath.Join(fComp.rootPath, main.Descriptor())
	if _, err := os.Stat(descPath); err == nil {
		hasDescriptor = true
		cm.l.Printf("Parsing references in descriptor of component %s\n", main.ComponentId())
		refs, parent, err = main.ParseRefs(descPath, tplC)
		if err != nil {
			return nil, err
		}
	} else {
		hasDescriptor = false
		cm.l.Printf("No descriptor in component: %s\n", main.ComponentId())
	}

	// Go through parents recursively
	if parent != nil {
		fM, err = cm.Init(parent, tplC)
		if err != nil {
			return nil, err
		}
	}

	// Parse the component model if any and merge it
	if hasDescriptor {
		cM, err := main.ParseModel(descPath, tplC)
		if err != nil {
			return nil, err
		}
		if fM != nil {
			fM, err = fM.Merge(cM)
			if err != nil {
				return nil, err
			}
		} else {
			fM = cM
		}
	}

	// Go through declared components
	for _, ref := range refs {
		c, err := ref.Component(fM)
		if err != nil {
			return nil, err
		}
		cM, err := cm.Init(c, tplC)
		if err != nil {
			return nil, err
		}
		if cM != nil {
			if fM != nil {
				fM, err = fM.Merge(cM)
				if err != nil {
					return nil, err
				}
			} else {
				fM = cM
			}
		}
	}

	return fM, nil
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

//HasDescriptor returns true if the fetched component contains a descriptor
func (fc fetchedComponent) hasDescriptor() bool {
	if fc.component.Descriptor() == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(fc.rootPath, fc.component.Descriptor())); err == nil {
		return true
	}
	return false
}

func (cm *componentManager) isComponentFetched(id string) (val fetchedComponent, present bool) {
	val, present = cm.fComps[id]
	return
}

func (cm *componentManager) fetchComponent(c Component) (fetchedComponent, error) {
	cm.l.Printf("Fetching component %s", c.ComponentId())
	fComp, isFetched := cm.isComponentFetched(c.ComponentId())
	if !isFetched {
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
