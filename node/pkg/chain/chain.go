// Package chain holds helpers for working with the infra-injected chain value.
package chain

// Normalize maps the infra-injected chain value to the chain name that
// config.orakl.network serves after the chain rename: cypress->mainnet,
// baobab->kairos. mainnet/kairos pass through unchanged so callers keep
// working both before and after infra flips the injected CHAIN value.
//
// It applies only to served config-bundle names. DB/subgraph schema names are
// still cypress/baobab in the database and must NOT be run through this.
func Normalize(chain string) string {
	switch chain {
	case "cypress":
		return "mainnet"
	case "baobab":
		return "kairos"
	default:
		return chain
	}
}
