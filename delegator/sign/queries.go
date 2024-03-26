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

	UpdateTransaction = `
		UPDATE transactions SET
			timestamp = @timestamp, "from" = @from, "to" = @to, input = @input, gas = @gas, value = @value, "chainId" = @chainId, "gasPrice" = @gasPrice,
			nonce = @nonce, v = @v, r = @r, s = @s, "rawTx" = @rawTx, "signedRawTx" = @signedRawTx, succeed = @succeed, function_id = @functionId,
			contract_id = @contractId, reporter_id = @reporterId
		WHERE
			transaction_id = @id
		RETURNING *;
	`
	GetContractByAddress   = `SELECT * FROM contracts WHERE address = @address;`
	GetReporterByFromAndTo = `
		SELECT *
		FROM reporters
		WHERE id IN (
			SELECT "B"
			FROM "_ContractToReporter"
			WHERE "A" IN (
				SELECT contract_id
				FROM contracts
				WHERE address = @to
			)
		)
		AND address = @from;
	`
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
	GetTransactions       = `SELECT * FROM transactions;`
	GetTransactionById    = `SELECT * FROM transactions WHERE transaction_id = @id;`
	DeleteTransactionById = `DELETE FROM transactions WHERE transaction_id = @id RETURNING *;`
)
