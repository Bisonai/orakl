package vrf

const (
	InsertVrf = `
	INSERT INTO vrf_keys (sk, pk, pk_x, pk_y, key_hash, chain_id)
	VALUES (@sk, @pk, @pk_x, @pk_y, @key_hash, @chain_id)
	RETURNING  vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash,
		(SELECT name FROM chains WHERE chains.chain_id = vrf_keys.chain_id) AS chain_name;
	`

	GetVrf = `
	SELECT vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash, chains.name AS chain_name
	FROM vrf_keys
	JOIN chains ON vrf_keys.chain_id = chains.chain_id
	WHERE vrf_keys.chain_id = @chain_id;
	`

	GetVrfWithoutChainId = `
	SELECT vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash, chains.name AS chain_name
	FROM vrf_keys
	JOIN chains ON vrf_keys.chain_id = chains.chain_id;
	`

	GetVrfById = `
	SELECT  vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash, chains.name AS chain_name
	FROM vrf_keys 
	JOIN chains ON vrf_keys.chain_id = chains.chain_id
	WHERE vrf_key_id = @id LIMIT 1;
	`

	UpdateVrfById = `
		UPDATE vrf_keys
		SET sk = @sk, pk = @pk, pk_x = @pk_x, pk_y = @pk_y, key_hash = @key_hash
		WHERE vrf_key_id = @id
		RETURNING  vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash,
		(SELECT name FROM chains WHERE chains.chain_id = vrf_keys.chain_id) AS chain_name;
	`

	DeleteVrfById = `
		DELETE FROM vrf_keys WHERE vrf_key_id = @id RETURNING  vrf_keys.vrf_key_id, vrf_keys.sk, vrf_keys.pk, vrf_keys.pk_x, vrf_keys.pk_y, vrf_keys.key_hash,
		(SELECT name FROM chains WHERE chains.chain_id = vrf_keys.chain_id) AS chain_name;
	`
)
