type Handler<T> = (payload: T) => void

export function createTaskMonitorSSE<T>(url: string, eventName: string, handler: Handler<T>) {
  const source = new EventSource(url)
  source.addEventListener(eventName, (event) => {
    handler(JSON.parse((event as MessageEvent).data) as T)
  })
  return source
}
