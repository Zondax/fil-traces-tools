# fil-trace-check

A comprehensive validation tool for Filecoin blockchain traces and state. This tool provides various validation commands to ensure the integrity of Filecoin blockchain data, including trace validation, address balance verification, and multisig state checking.

## Features

### Range-based Trace Validation
Validate traces over any range of epochs using `--start` and `--end` flags:
- **Canonical Chain Validation**: Ensures the integrity of the traces by verifying miners match on-chain data.
- **JSON Validation**: Validates the JSON structure of trace data
- **Null Blocks Validation**: Verifies null blocks in the traces are null blocks on chain.
- **Sequential Address Balance Validation**: Validates balances of addresses in the traces match on-chain balances at epochs with address activity.
- **Sequential Multisig State Validation**: Validates state changes of multisig addresses in the traces match on-chain state at epochs with multisig events.

### Address-based Validation
Two approaches for validating address-related data:

#### Event-based Validation
- **Address Balance Validation**: Validates balances at epochs with activity using event providers
- **Multisig State Validation**: Tracks state changes at epochs with multisig events

#### Sequential Validation
- **Address Balance Sequential**: Processes every epoch in a range and finds activity for addresses in the traces.
- **Multisig State Sequential**: Validates state changes across all epochs in a range

### Reporting
- **Generate Report**: Export validation results for any check type as JSON

## Prerequisites

- Go 1.24.4 or higher
- Access to a Filecoin node (RPC endpoint)
- S3-compatible storage for trace data
- Configuration file (`config.yaml`)

## Installation

```bash
go install github.com/zondax/fil-trace-check@latest
```

## Configuration

Create a `config.yaml` file in the project root with the following structure:

```yaml
# Network configuration
network_name: "mainnet"  # Network name (mainnet, calibration, etc.)
network_symbol: "FIL"
node_url: "https://api.node.glif.io/rpc/v1"  # Filecoin node RPC URL
node_token: ""  # Optional: Node authentication token

# S3 storage configuration (for trace data)
s3_url: ""  # S3 endpoint URL
s3_ssl: true  # Use SSL for S3 connection
s3_access_key: ""  # S3 access key
s3_secret_key: ""  # S3 secret key
s3_bucket: ""  # S3 bucket name
s3_raw_data_path: ""  # Path within bucket for raw data
```

## Usage

### Basic Command Structure

```bash
fil-trace-check <command> [flags]
```

### Available Commands

#### 1. Validate Null Blocks

Validates that null blocks in traces match the actual on-chain state.

```bash
fil-trace-check validate-null-blocks --start <start_epoch> --end <end_epoch> --db-path <path>
```

Flags:
- `--start`: Starting epoch number (default: 1)
- `--end`: Ending epoch number (default: 100)
- `--db-path`: Path to store validation progress database (default: ".")

Example:
```bash
fil-trace-check validate-null-blocks --start 387926 --end 387930 --db-path ./validation-db
```

#### 2. Validate JSON

Validates the JSON structure of trace data for a range of epochs.

```bash
fil-trace-check validate-json --start <start_epoch> --end <end_epoch> --db-path <path>
```

Flags:
- `--start`: Starting epoch number (default: 1)
- `--end`: Ending epoch number (default: 100)
- `--db-path`: Path to store validation progress database (default: ".")

Example:
```bash
fil-trace-check validate-json --start 387926 --end 387930
```

#### 3. Validate Canonical Chain

Validates the integrity of the canonical chain by verifying miners match on-chain data.

```bash
fil-trace-check validate-canonical-chain --start <start_epoch> --end <end_epoch> --db-path <path>
```

Flags:
- `--start`: Starting epoch number (default: 1)
- `--end`: Ending epoch number (default: 100)
- `--db-path`: Path to store validation progress database (default: ".")

#### 4. Validate Address Balance (Event-based)

Validates address balances at epochs with activity using event providers to get the epochs where each address has activity.

```bash
fil-trace-check validate-address-balance --address-file <path> --db-path <path> --event-provider <provider> --event-provider-token <token>
```

Flags:
- `--address-file`: Path to a newline-separated file containing addresses to check
- `--db-path`: Path to store validation progress database (default: ".")
- `--event-provider`: Event provider to use (default: "beryx")
- `--event-provider-token`: Optional event provider authentication token

Example address file:
```
f1abcdefghijklmnopqrstuvwxyz123456
f3abcdefghijklmnopqrstuvwxyz123456
```

The validation process:
1. For each address, queries event provider for epochs with activity
2. Applies all transactions at each active epoch to track balance changes
3. Verifies balances never go negative
4. Compares calculated balances with on-chain balances

#### 5. Validate Address Balance Sequential

Validates address balances across every epoch in a range.

```bash
fil-trace-check validate-address-balance-sequential --address-file <path> --start <start_epoch> --end <end_epoch> --db-path <path>
```

Flags:
- `--address-file`: Path to a newline-separated file containing addresses to check
- `--start`: Starting epoch number (default: 1, optional)
- `--end`: Ending epoch number (required)
- `--db-path`: Path to store validation progress database (default: ".")

Example:
```bash
fil-trace-check validate-address-balance-sequential --address-file addresses.txt --start 1 --end 1000
```

#### 6. Validate Multisig State (Event-based)

Validates multisig wallet state transitions at epochs with events.

```bash
fil-trace-check validate-multisig-state --address-file <path> --db-path <path> --event-provider <provider> --event-provider-token <token>
```

Flags:
- `--address-file`: Path to a newline-separated file containing multisig addresses
- `--db-path`: Path to store validation progress database (default: ".")
- `--event-provider`: Event provider to use (default: "beryx")
- `--event-provider-token`: Optional event provider authentication token

#### 7. Validate Multisig State Sequential

Validates multisig wallet state across every epoch in a range. 

```bash
fil-trace-check validate-multisig-state-sequential --address-file <path> --start <start_epoch> --end <end_epoch> --db-path <path>
```

Flags:
- `--address-file`: Path to a newline-separated file containing multisig addresses
- `--start`: Starting epoch number (default: 1, optional)
- `--end`: Ending epoch number (required)
- `--db-path`: Path to store validation progress database (default: ".")

The validation process:
1. For each multisig address, processes all epochs in the range
2. Tracks state changes including signers, locked balance, and unlock duration
3. Compares parsed state with on-chain state at each epoch

## Progress Tracking

All validation commands store their progress in a local BoltDB database. This allows:
- Resuming validation from where it left off
- Tracking validation status for each epoch or event
- Storing error messages for failed validations

Progress data is stored in the path specified by `--db-path` with a bucket name specific to each validation type.

## Generate Report

Generate a JSON report for any validation check, exporting all results from the database.

```bash
fil-trace-check generate-report --check <check> --db-path <path> --report-path <path>
```

Flags:
- `--check`: Check to generate report for (required). Possible values: 
  - `validate-null-blocks`
  - `validate-json`
  - `validate-canonical-chain`
  - `validate-address-balance`
  - `validate-multisig-state`
  - `validate-address-balance-sequential`
  - `validate-multisig-state-sequential`
- `--db-path`: Path to validation progress database (default: ".")
- `--report-path`: Path to store report (default: ".")

Example:
```bash
fil-trace-check generate-report --check validate-address-balance --db-path ./validation-db --report-path ./reports
```

The generated report filename follows the pattern: `<check>_<timestamp>.json`

## Choosing Between Sequential and Event-based Validation

For address balance and multisig state validation, you have two options:

### Event-based Validation
Best for:
- Addresses with sparse activity
- Quick validation of specific events
- When you have access to an event provider

### Sequential Validation  
Best for:
- Comprehensive validation across all epochs
- When you don't have event provider access
- Validating continuous ranges of activity

## Error Handling

- Each epoch's validation status is tracked independently
- Errors are logged with detailed messages
- Failed validations don't stop the entire process
- Progress is saved after each epoch