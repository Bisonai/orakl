import { fetchData, aggregateData } from '../job/job.utils';
import { IAdapter } from '../job/job.types';

// runs job through local adapter json file
// example: npx ts-node ./src/tools/sample-run-adapter.ts ./ada-usdt.adapter.json

const main = async () => {
    const adapterJsonPath = process.argv.slice(2)[0];

    const adapter: IAdapter = require(adapterJsonPath);
    const adapterDefintions = adapter.feeds.map((feed)=> {
      return feed.definition;
    })
   
    const data = await fetchData(adapterDefintions, console);
    console.log(data);

    const aggregate = aggregateData(data);
    console.log(aggregate);
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})