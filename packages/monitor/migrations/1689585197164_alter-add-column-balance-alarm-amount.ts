/* eslint-disable @typescript-eslint/naming-convention */
import { MigrationBuilder, ColumnDefinitions } from 'node-pg-migrate';

export const shorthands: ColumnDefinitions | undefined = undefined;

export async function up(pgm: MigrationBuilder): Promise<void> {
  pgm.addColumn('account', {
    balance_alarm_amount: {
      type: 'int',
      notNull: true,
      default: 0
    }
  }, {
    ifNotExists: true
  });
  
}

export async function down(pgm: MigrationBuilder): Promise<void> {
}
