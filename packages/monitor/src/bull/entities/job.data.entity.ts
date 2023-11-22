export class JobCompleted {
  service?: string;
  name?: string;
  job_id?: string;
  job_name?: string;
  contract_address?: string;
  block_number?: string;
  block_hash?: string;
  callback_address?: string;
  block_num?: number;
  request_id?: string;
  acc_id?: string;
  pk?: Array<string>;
  seed?: string;
  proof?: Array<string>;
  u_point?: string;
  pre_seed?: string;
  num_words?: number;
  v_components?: Array<string>;
  callback_gas_limit?: string;
  sender?: string;
  is_direct_payment?: string;
  event?: JSON;
  data?: string;
  data_set?: JSON;
  added_at?: number;
  process_at?: number;
  completed_at?: number;
}

export class JobFailed extends JobCompleted {
  error?: Array<string>;
}
