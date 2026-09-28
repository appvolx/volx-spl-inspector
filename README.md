# volx-spl-inspector

**VOLX Ecosystem Tool** — Local SPL token & mint inspector for Solana.

Part of the [VOLX](https://volx.app) tooling suite.  
GitHub org: [https://github.com/appvolx](https://github.com/appvolx)

## What it does

`volx-spl-inspector` is a **local-only Go CLI** that queries public Solana RPC endpoints to inspect:

- Mint accounts (supply, decimals, freeze/mint authority)
- Token accounts (balance, owner, delegated amount)
- Associated Token Account (ATA) derivation
- Basic Metaplex metadata pointers when present

It uses only public JSON-RPC methods (`getAccountInfo`, `getTokenSupply`, `getTokenAccountsByOwner`).  
No private keys, no signing, no credentials required.

## Why it exists

VOLX integrates Pump.fun launches and Raydium liquidity. Operators and integrators frequently need a quick, scriptable way to verify mint parameters and token account state without opening a block explorer or running a full wallet. This tool fills that gap.

## Features

- Pure Go, single static binary possible
- Base58 address validation
- ATA derivation (associated token program)
- Human-readable and JSON output
- Configurable RPC endpoint (defaults to public mainnet)
- No third-party Solana SDK required for core paths (uses raw RPC + encoding helpers)

## Requirements

- Go ≥ 1.21
- Internet access to a Solana RPC (public or your own)

## Installation

```bash
cd volx-spl-inspector
go build -o volx-spl-inspector ./cmd/volx-spl-inspector
# or
go install ./cmd/volx-spl-inspector
```

## Quick start

```bash
# Inspect a mint (example: Wrapped SOL)
./volx-spl-inspector mint So11111111111111111111111111111111111111112

# Inspect a token account
./volx-spl-inspector account <TOKEN_ACCOUNT_PUBKEY>

# Derive ATA for owner + mint
./volx-spl-inspector ata --owner <WALLET> --mint <MINT>

# Custom RPC
./volx-spl-inspector mint <MINT> --rpc https://api.mainnet-beta.solana.com

# JSON
./volx-spl-inspector mint <MINT> --json
```

## Security model

- Read-only RPC calls only
- Never accepts or stores private keys / seed phrases
- Suitable for public repositories
- All traffic is outbound HTTPS to the RPC you choose

## Integration with VOLX

Useful when debugging Pump.fun created mints or Raydium pool vault token accounts.  
Cross-check results against Solscan / the VOLX launch and portfolio UIs.

## License

MIT — complementary tooling for the VOLX project (https://volx.app).
