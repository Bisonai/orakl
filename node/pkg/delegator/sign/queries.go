package sign

const (
	GetFeePayers      = `SELECT * FROM fee_payers;`
	InsertTransaction = `
		INSERT INTO transactions (
			timestamp, "from", "to", input, gas, value, "chainId", "gasPrice", nonce, v, r, s, "rawTx", "signedRawTx", "succeed", function_id, contract_id, reporter_id
		) VALUES (
			@timestamp, @from, @to, @input, @gas, @value, @chainId, @gasPrice, @nonce, @v, @r, @s, @rawTx, @signedRawTx, @succeed, @functionId, @contractId, @reporterId
		) RETURNING *;
	`

	GetContractByAddress          = `SELECT * FROM contracts WHERE address = @address;`
	GetFunctionByToAndEncodedName = `
		SELECT *
		FROM functions
		WHERE contract_id IN (
			SELECT contract_id
			FROM contracts
			WHERE address = @to
		)
		AND "encodedName" = @encodedName;
	`
	GetTransactions       = `SELECT * FROM transactions ORDER BY transaction_id DESC LIMIT @limit OFFSET @offset;`
	GetTransactionById    = `SELECT * FROM transactions WHERE transaction_id = @id;`
	DeleteTransactionById = `DELETE FROM transactions WHERE transaction_id = @id RETURNING *;`
)
