// Package plugin is the internal facade used by Spire commands to discover
// and invoke plugins. Types are aliased from plugin/sdk so that the public
// SDK and the internal plumbing always stay in sync without duplication.
package plugin

import sdk "github.com/schaemi85/spire/plugin/sdk"

// Type aliases — every cmd package that imports this package continues to
// reference plugin.HookContext, plugin.HookName, etc. without changes.
type (
	HookName    = sdk.HookName
	HookContext = sdk.HookContext
	ServiceInfo = sdk.ServiceInfo
	HookResult  = sdk.HookResult
)

// Hook name constants forwarded from the SDK.
const (
	HookBeforeAddService HookName = sdk.HookBeforeAddService
	HookAfterAddService  HookName = sdk.HookAfterAddService
	HookBeforeUpgrade    HookName = sdk.HookBeforeUpgrade
	HookAfterUpgrade     HookName = sdk.HookAfterUpgrade
)

// AllHookNames lists every supported hook in execution order.
var AllHookNames = sdk.AllHooks
