export class PubSubStop {
  subscriber
  publisher

  constructor(redisClient) {
    this.subscriber = redisClient.duplicate()
    this.publisher = redisClient.duplicate()
  }

  /**
   * Connect to publisher and subscriber. It must be called before the
   * pub/sub communication channel is utilized. Connection is a
   * asynchronous, and can't be included in constructor.
   */
  async init() {
    await this.subscriber.connect()
    await this.publisher.connect()
  }

  /**
   * Stop channel name getter.
   */
  getChannelName(id: string) {
    return `listener:stop:${id}`
  }

  /**
   * Subscribe to channel denoted as `channelName`. If any message
   * comes through that channel, stop the listener and unsuscribe.
   *
   * @param {} identification of listener executed by `setInterval`
   * @param {string} listener ID
   */
  async setupSubscriber(listenerId, id: string) {
    const channelName = this.getChannelName(id)
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const unsubscribeFn = async (message, channel) => {
      clearInterval(listenerId)
      await this.subscriber.unsubscribe(channelName)
    }
    await this.subscriber.subscribe(channelName, unsubscribeFn)
  }

  /**
   * Publish message to stop listener.
   *
   * @param {string} listener ID
   */
  async stop(id: string) {
    const channelName = this.getChannelName(id)
    await this.publisher.publish(channelName, 'stop')
  }
}
