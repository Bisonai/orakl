exports.shorthands = undefined;

exports.up = (pgm) => {
  pgm.createSequence("config_idx_seq");

  pgm.createTable("config", {
    idx: {
      type: "integer",
      notNull: true,
      primaryKey: true,
    },
    name: { type: "varchar", notNull: true },
    value: { type: "varchar", notNull: true },
    }, {
    primaryKey: "idx",
  });

  pgm.sql(
    `ALTER TABLE config ALTER COLUMN idx SET DEFAULT nextval('config_idx_seq'::regclass);`
  );
  
};


exports.down = (pgm) => {
  pgm.dropTable("config");

  pgm.dropSequence("config_id_seq");
};