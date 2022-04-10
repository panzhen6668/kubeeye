package pkg

// Install plugin param
var (
	PluginResource   = "plugins"
	MaxCheckPodCount = 15
	IntervalsTime    = 20
)

//plugin install status
const (
	PluginIntalled     string = "installed"
	PluginInstalling   string = "installing"
	PluginUninstalled  string = "uninstalled"
	PluginUninstalling string = "uninstalling"
)
