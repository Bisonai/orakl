export class AggregatorJobCompleted {
    service?: string;
    name?: string;
    job_id?: string;
    job_name?: string;
    oracle_address?: string;
    delay?: number;
    round_id?: number;
    worker_source?: string;;
    submission?: string;
    data_set?: JSON;
    added_at?: number;
    process_at?: number;
    completed_at?: number;
  }
  
  export class AggregatorJobFailed extends AggregatorJobCompleted {
    error?: Array<string>;
  }
  