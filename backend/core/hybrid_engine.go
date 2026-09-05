package core

type EngineKind string

const (
	EngineHTTP    EngineKind = "http"
	EngineBrowser EngineKind = "browser"
	EngineHybrid  EngineKind = "hybrid"
)

func ChooseEngine(needsBrowser bool, preferHTTP bool) EngineKind {
	if needsBrowser && preferHTTP {
		return EngineHybrid
	}
	if needsBrowser {
		return EngineBrowser
	}
	return EngineHTTP
}
