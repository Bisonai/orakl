/* eslint-disable camelcase */

exports.shorthands = undefined;

exports.up = (pgm) => {
  pgm.createTable("redis", {
    service: {
      type: "varchar",
    },
    host: {
      type: "varchar",
    },
    port: {
      type: "integer",
    },
  });
};

exports.down = (pgm) => {
  pgm.dropTable("redis");
};
