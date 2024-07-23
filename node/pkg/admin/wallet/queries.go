package wallet

const (
	GetWallets = `SELECT * FROM wallets;`

	InsertWallet = `INSERT INTO wallets (pk) VALUES (@pk) RETURNING *;`

	GetWalletById = `SELECT * FROM wallets WHERE id = @id;`

	UpdateWalletById = `
	UPDATE wallets
	SET pk = @pk
	WHERE id = @id
	RETURNING *;
	`

	DeleteWalletById = `
	DELETE FROM wallets WHERE id = @id RETURNING *;
	`
)
