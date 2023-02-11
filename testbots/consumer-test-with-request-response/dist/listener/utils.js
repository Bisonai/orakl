const FILE_NAME = import.meta.url;
export function validateListenerConfig(config, logger) {
    const requiredProperties = ['address', 'eventName'];
    for (const _config of config) {
        const propertyExist = requiredProperties.map((p) => (_config[p] ? true : false));
        const allPropertiesExist = propertyExist.every((i) => i);
        if (!allPropertiesExist) {
            logger?.error({ name: 'validateListenerConfig', file: FILE_NAME, ..._config }, '_config');
            return false;
        }
    }
    return true;
}
//# sourceMappingURL=utils.js.map