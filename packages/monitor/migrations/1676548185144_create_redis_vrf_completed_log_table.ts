/* eslint-disable camelcase */

exports.shorthands = undefined;

exports.up = (pgm) => {
  pgm.createTable("redis_vrf_completed", {
    service: { type: "varchar" },
    name: { type: "varchar" },
    job_id: { type: "varchar" },
    job_name: { type: "varchar" },
    contract_address: { type: "varchar" },
    block_number: { type: "varchar" },
    block_hash: { type: "varchar" },
    callback_address: { type: "varchar" },
    block_num: { type: "integer" },
    request_id: { type: "varchar" },
    acc_id: { type: "varchar" },
    pk: { type: "text[]" },
    seed: { type: "varchar" },
    proof: { type: "text[]" },
    u_point: { type: "varchar" },
    pre_seed: { type: "varchar" },
    num_words: { type: "integer" },
    v_components: { type: "text[]" },
    callback_gas_limit: { type: "numeric" },
    sender: { type: "varchar" },
    is_direct_payment: { type: "boolean" },
    event: { type: "jsonb" },
    data_set: { type: "jsonb" },
    data: { type: "text" },
    added_at: { type: "bigint" },
    process_at: { type: "bigint" },
    completed_at: { type: "bigint" },
  });
};

exports.down = (pgm) => {
  pgm.dropTable("redis_vrf_completed");
};
