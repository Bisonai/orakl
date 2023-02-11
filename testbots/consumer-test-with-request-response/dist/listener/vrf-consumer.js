import { Event } from "./event";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";
const abis = await readTextFile("./src/abis/consumer.json");
export function buildVrfListener(config) {
    new Event(processConsumerEvent, abis, config).listen();
}
function processConsumerEvent(iface) {
    async function wrapper(log) {
        const eventData = iface.parseLog(log).args;
        let jsonResult = [];
        const jsonPath = "./tmp/listener/consumer-fulfill-log.json";
        if (!existsSync(jsonPath))
            await writeTextFile(jsonPath, JSON.stringify(jsonResult));
        const data = await readTextFile(jsonPath);
        if (data)
            jsonResult = JSON.parse(data);
        let result = {};
        if (eventData) {
            result = {
                requestId: eventData.requestId.toString(),
                randomWords: eventData.randomWords.map((r) => {
                    return r.toString();
                }),
            };
            jsonResult.push(result);
        }
        console.debug("processVrfEvent:data", jsonResult.length);
        await writeTextFile(jsonPath, JSON.stringify(jsonResult));
    }
    return wrapper;
}
async function main() {
    const listenersConfig = {
        address: process.env.VRF_CONSUMER ?? '',
        eventName: "RandomWordsFulfilled",
    };
    console.log(listenersConfig);
    const config = listenersConfig;
    buildVrfListener(config);
}
main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
//# sourceMappingURL=vrf-consumer.js.map