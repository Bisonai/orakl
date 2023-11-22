/* eslint-disable camelcase */

exports.shorthands = undefined;

exports.up = pgm => {
    pgm.createTable('account', {
      address: {
        type: 'text',
        notNull: true,
        primaryKey: true
      },
      name: {
        type: 'text',
        notNull: true
      },       
      type: {
        type: 'text',
        notNull: true
      }
    })
    pgm.createIndex('account', 'address')
    pgm.createTable('balance', {
        address: {
          type: 'text',
          notNull: true,
          references: '"account"',
          onDelete: 'cascade'
        },
        balance: {
          type: 'text',
          notNull: true
        },       
        time: {
          type: 'timestamp',
          notNull: true,
          default: pgm.func('current_timestamp')
        }             
      })    
      pgm.createIndex('balance', ['address', 'time'])      
};

exports.down = pgm => {
    pgm.dropTable('balance')
    pgm.dropTable('account')
};
