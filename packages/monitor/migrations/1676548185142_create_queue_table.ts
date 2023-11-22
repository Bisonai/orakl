/* eslint-disable camelcase */

exports.shorthands = undefined;

exports.up = (pgm) => {
  pgm.createSequence('queue_idx_seq', { ifNotExists: true });
  pgm.createTable("queue", {
    idx: {
      type: "integer",
      notNull: true,
      primaryKey: true,
    },
    service: {
      type: "varchar",
    },
    name: {
      type: "varchar",
    },
    status: {
      type: "bool",
      default: true,
    },    
  });
  pgm.sql(
    `ALTER TABLE queue ALTER COLUMN idx SET DEFAULT nextval('queue_idx_seq'::regclass);`
  );
};

exports.down = (pgm) => {
  pgm.dropTable('queue');
  pgm.dropSequence('queue_idx_seq');
};
