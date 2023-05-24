exports.shorthands = undefined;

exports.up = (pgm) => {
  pgm.createTable("redis_aggregator_completed", {
    service: { type: "varchar" },
    name: { type: "varchar" },
    job_id: { type: "varchar" },
    job_name: { type: "varchar" },
    oracle_address: { type: "varchar" },
    delay: { type: "integer" },
    round_id: { type: "integer" },
    worker_source: { type: "varchar" },
    submission: { type: "varchar" },
    data_set: { type: "jsonb" },
    added_at: { type: "bigint" },
    process_at: { type: "bigint" },
    completed_at: { type: "bigint" },
  });
  
};

exports.down = (pgm) => {
  pgm.dropTable("redis_aggregator_completed");
};