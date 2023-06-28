import { registerAs } from '@nestjs/config';

export const jwtConstants = {
  secret: process.env.JWT_SECRET,
};
export const PASSWORD = process.env.PASSWORD;

export const commonConfig = registerAs("common", () => ({
  provider: process.env.PROVIDER,
}));

export const databaseConfig = registerAs("database", () => ({
  monitor: {
    user: process.env.MONITOR_POSTGRES_USER,
    host: process.env.MONITOR_POSTGRES_HOST,
    database: process.env.MONITOR_POSTGRES_DATABASE,
    password: process.env.MONITOR_POSTGRES_PASSWORD,
    port: parseInt(process.env.MONITOR_POSTGRES_PORT, 10) || 5432,
  },
  orakl: {
    user: process.env.ORAKL_POSTGRES_USER,
    host: process.env.ORAKL_POSTGRES_HOST,
    database: process.env.ORAKL_POSTGRES_DATABASE,
    password: process.env.ORAKL_POSTGRES_PASSWORD,
    port: parseInt(process.env.ORAKL_POSTGRES_PORT, 10) || 5432,
  },
  graphNode: {
    user: process.env.GRAPH_NODE_POSTGRES_USER,
    host: process.env.GRAPH_NODE_POSTGRES_HOST,
    database: process.env.GRAPH_NODE_POSTGRES_DATABASE,
    password: process.env.GRAPH_NODE_POSTGRES_PASSWORD,
    port: parseInt(process.env.GRAPH_POSTGRES_PORT, 10) || 5432,
  },
}));
