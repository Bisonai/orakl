package wallet

const (
	GetWallets = `SELECT * FROM wallets;`

	InsertWallet = `INSERT INTO wallets (pk, address) VALUES (@pk, @address) RETURNING *;`

	GetWalletById = `SELECT * FROM wallets WHERE id = @id;`

	UpdateWalletById = `
	UPDATE wallets
	SET pk = @pk, address = @address
	WHERE id = @id
	RETURNING *;
	`

	DeleteWalletById = `
	DELETE FROM wallets WHERE id = @id RETURNING *;
	`
)
