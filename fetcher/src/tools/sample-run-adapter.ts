import axios, { AxiosRequestConfig } from 'axios';
import { IAdapter } from '../job/job.types';
import { DATA_FEED_REDUCER_MAPPING } from '../job/job.reducer'
import { FetcherError, FetcherErrorCode } from '../job/job.errors'



function buildReducer(reducerMapping, reducers) {
    return reducers.map((r) => {
      const reducer = reducerMapping[r.function]
      if (!reducer) {
        throw new FetcherError(FetcherErrorCode.InvalidReducer)
      }
      return reducer(r?.args)
    })
  }
  
  // https://medium.com/javascript-scene/reduce-composing-software-fe22f0c39a1d
export const pipe =
    (...fns) =>
    (x) =>
      fns.reduce((v, f) => f(v), x)

function checkDataFormat(data) {
    if (!data) {
        // check if priceFeed is null, undefined, NaN, "", 0, false
        throw new FetcherError(FetcherErrorCode.InvalidDataFeed)
    } else if (!Number.isInteger(data)) {
        // check if priceFeed is not Integer
        throw new FetcherError(FetcherErrorCode.InvalidDataFeedFormat)
    }
}


const fetchCallTest = async () => {
    const adapterList: IAdapter = require("./ada-usdt.adapter.json");
    const adapterListFeeds = adapterList.feeds;
    const data = await Promise.allSettled(
        adapterListFeeds.map(async (adapter) => {
            const url = adapter.definition.url;
            const method = adapter.definition.method;
            const headers = adapter.definition.headers;
      
            const requestConfig: AxiosRequestConfig = {};
            requestConfig.method = method;
            requestConfig.headers = {...headers};
    
            const rawDatum = (await (axios.get(url, requestConfig))).data;
    
            try {
                // FIXME Build reducers just once and use. Currently, can't
                // be passed to queue, therefore has to be recreated before
                // every fetch.
                const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, adapter.definition.reducers)
                const datum = pipe(...reducers)(rawDatum)
                checkDataFormat(datum)
                return {value: datum}
                // console.log({value: datum});
                // return { id: adapter.id, value: datum }
              } catch (e) {
    
                throw e
              }
        })
    )
    console.log(data.flatMap((D) => (D.status == 'fulfilled' ? [D.value] : [])));
}

fetchCallTest();
