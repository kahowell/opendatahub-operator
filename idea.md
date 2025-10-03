```go
type Components struct {
    // ... existing fields omitted

	// Langfuse component configuration
	Langfuse DSCLangfuse `json:"langfuse,omitempty"`
}

type HelmManagedComponentRegistry map[string]HelmManagedComponent

func (r HelmManagedComponentRegistry) Register(name string, valuesFactory func(spec DataScienceClusterSpec) (chartutil.Values, error), watches []schema.GroupVersionKind) {
	chrt, err := loader.LoadArchive(nil)
	if err != nil {
		panic(err)
	}
	r[name] = HelmManagedComponent{
		Chart:    chrt,
		ValuesFn: valuesFactory,
		Watches:  watches,
	}
}

func (r HelmManagedComponentRegistry) Render(name string, spec DataScienceClusterSpec) (map[string]string, error) {
	component := r[name]
	// TODO cache during registration
	for _, file := range component.Chart.Files {
		if file.Name == "values.rhoai.yaml" {
			// TODO read and merge with values from ValuesFn
		}
	}
	values, err := component.ValuesFn(spec)
	if err != nil {
		return nil, err
	}
	values, err = chartutil.MergeValues(component.Chart, values)
	if err != nil {
		return nil, err
	}
	return engine.Render(component.Chart, values)
}

var HelmManagedComponents HelmManagedComponentRegistry

// +odh:helm:chart=langfuse
// +odh:watch=Deployment
// +odh:watch=ServiceAccount
// +odh:watch=core/v1/Pod
type DSCLangfuse struct {
	// Fields common across components
	common.ManagementSpec `json:",inline"`

	// Fields specific to the component
	LangfuseFeatures `json:"features,omitempty"`
}

type LangfuseFeatures struct {
	// +odh:helm:value_path=langfuse.features.experimentalFeaturesEnabled
	ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled"`
}

type HelmManagedComponent struct {
	Chart    *chart.Chart
	ValuesFn func(DataScienceClusterSpec) (chartutil.Values, error)
	Watches  []schema.GroupVersionKind
}

func init() {
	HelmManagedComponents.Register("langfuse", LangfuseValues, []schema.GroupVersionKind{schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}})
}

// TODO write controller factory that creates a separate controller instance for each helm-managed operator

func LangfuseValues(spec DataScienceClusterSpec) (chartutil.Values, error) {
	values := chartutil.Values{}
	if err := strvals.ParseInto(fmt.Sprintf("%s=%s", "langfuse.features.experimentalFeaturesEnabled", strconv.FormatBool(spec.Components.Langfuse.LangfuseFeatures.ExperimentalFeaturesEnabled)), values); err != nil {
		return nil, err
	}
	return values, nil
}
```
