// volx-spl-inspector — local SPL mint / token account inspector for VOLX.
// Client-side only. Public RPC. No credentials.
package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultRPC   = "https://api.mainnet-beta.solana.com"
	userAgent   = "volx-spl-inspector/1.0 (+https://volx.app)"
	tokenProgram = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	ataProgram   = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "mint":
		runMint(os.Args[2:])
	case "account":
		runAccount(os.Args[2:])
	case "ata":
		runATA(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `volx-spl-inspector — VOLX ecosystem tool (https://volx.app)

Usage:
  volx-spl-inspector mint <MINT_PUBKEY> [--rpc URL] [--json]
  volx-spl-inspector account <TOKEN_ACCOUNT> [--rpc URL] [--json]
  volx-spl-inspector ata --owner <WALLET> --mint <MINT> [--rpc URL] [--json]

Examples:
  volx-spl-inspector mint So11111111111111111111111111111111111111112
  volx-spl-inspector ata --owner <WALLET> --mint <MINT>
`)
}

func runMint(args []string) {
	fs := flag.NewFlagSet("mint", flag.ExitOnError)
	rpc := fs.String("rpc", defaultRPC, "Solana RPC URL")
	asJSON := fs.Bool("json", false, "JSON output")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "mint address required")
		os.Exit(2)
	}
	mint := fs.Arg(0)

	data, err := getAccountInfo(*rpc, mint)
	if err != nil {
		fatal(err)
	}
	if data == nil {
		fatal(fmt.Errorf("mint account not found: %s", mint))
	}

	info, err := parseMint(data)
	if err != nil {
		fatal(err)
	}
	info["address"] = mint
	info["rpc"] = *rpc

	printResult(info, *asJSON)
}

func runAccount(args []string) {
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	rpc := fs.String("rpc", defaultRPC, "Solana RPC URL")
	asJSON := fs.Bool("json", false, "JSON output")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "token account address required")
		os.Exit(2)
	}
	addr := fs.Arg(0)

	data, err := getAccountInfo(*rpc, addr)
	if err != nil {
		fatal(err)
	}
	if data == nil {
		fatal(fmt.Errorf("account not found: %s", addr))
	}

	info, err := parseTokenAccount(data)
	if err != nil {
		fatal(err)
	}
	info["address"] = addr
	info["rpc"] = *rpc

	printResult(info, *asJSON)
}

func runATA(args []string) {
	fs := flag.NewFlagSet("ata", flag.ExitOnError)
	rpc := fs.String("rpc", defaultRPC, "Solana RPC URL")
	owner := fs.String("owner", "", "Owner wallet")
	mint := fs.String("mint", "", "Mint address")
	asJSON := fs.Bool("json", false, "JSON output")
	_ = fs.Parse(args)

	if *owner == "" || *mint == "" {
		fmt.Fprintln(os.Stderr, "--owner and --mint required")
		os.Exit(2)
	}

	// Note: full ATA derivation needs ed25519 PDA logic.
	// We surface the inputs and instruct the user; a pure-Go PDA
	// implementation would pull in crypto libraries. For this lightweight
	// tool we call getProgramAccounts-style path is heavy, so we document
	// the standard seeds and optionally verify if the account exists when
	// the user already knows the ATA.
	info := map[string]any{
		"owner":         *owner,
		"mint":          *mint,
		"token_program": tokenProgram,
		"ata_program":   ataProgram,
		"note":          "ATA = PDA([owner, token_program, mint], ata_program). Use Solana CLI `spl-token address` or web3.js getAssociatedTokenAddressSync for exact derivation offline.",
		"rpc":           *rpc,
	}
	printResult(info, *asJSON)
}

// --- RPC helpers ---

func getAccountInfo(rpc, pubkey string) ([]byte, error) {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getAccountInfo",
		"params": []any{
			pubkey,
			map[string]string{"encoding": "base64"},
		},
	}
	raw, err := postJSON(rpc, body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Result *struct {
			Value *struct {
				Data []string `json:"data"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", resp.Error.Message)
	}
	if resp.Result == nil || resp.Result.Value == nil || len(resp.Result.Value.Data) < 1 {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(resp.Result.Value.Data[0])
}

func postJSON(url string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// --- Binary parsers (SPL Token layout) ---

func parseMint(data []byte) (map[string]any, error) {
	// Mint layout (at least 82 bytes)
	if len(data) < 82 {
		return nil, fmt.Errorf("mint data too short: %d bytes", len(data))
	}
	mintAuthorityOption := binary.LittleEndian.Uint32(data[0:4])
	supply := binary.LittleEndian.Uint64(data[36:44])
	decimals := data[44]
	isInitialized := data[45] == 1
	freezeAuthorityOption := binary.LittleEndian.Uint32(data[46:50])

	out := map[string]any{
		"decimals":       decimals,
		"supply_raw":     supply,
		"is_initialized": isInitialized,
		"has_mint_authority":   mintAuthorityOption == 1,
		"has_freeze_authority": freezeAuthorityOption == 1,
	}
	return out, nil
}

func parseTokenAccount(data []byte) (map[string]any, error) {
	// Token account layout ~165 bytes
	if len(data) < 72 {
		return nil, fmt.Errorf("token account data too short: %d bytes", len(data))
	}
	// mint: 32 bytes at 0
	// owner: 32 bytes at 32
	// amount: u64 at 64
	amount := binary.LittleEndian.Uint64(data[64:72])
	out := map[string]any{
		"amount_raw": amount,
		"mint_bytes_prefix": fmt.Sprintf("%x", data[0:8]),
		"owner_bytes_prefix": fmt.Sprintf("%x", data[32:40]),
		"note": "Full base58 mint/owner decoding requires a base58 encoder; prefixes shown for verification.",
	}
	return out, nil
}

func printResult(v any, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
