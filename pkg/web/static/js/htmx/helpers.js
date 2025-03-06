/*
Checks the htmx requestConfig from the event detail (if any) and determines if the
path and method match what the request was for; allowing you to disambiguate between
different htmx requests on the same page.
*/
export function isRequestFor(e, path, method) {
  // Check the request config for the path and method if it has been configured.
  const config = e.detail?.requestConfig;
  if (config) {
    return config.path === path && config.verb === method;
  }

  // Check the detail directly if this is during a request config event.
  if (e.detail?.path && e.detail?.verb) {
    return e.detail.path === path && e.detail.verb === method;
  }

  // Otherwise return false since we can't determine the configuration.
  return false;
}

/*
Like isRequestFor but uses a regular expression to match the path of the request. This
is useful for matching a group of requests that share a common path but have different
(such as paths that have UUIDs for example).
*/
export function isRequestMatch(e, pattern, method) {
  if (typeof(pattern) === 'string') {
    pattern = new RegExp(pattern);
  }

  if (!pattern instanceof RegExp) {
    throw new Error('request pattern for the path must be a string or RegExp');
  }

  const config = e.detail?.requestConfig;
  if (config) {
    return config.verb === method && pattern.test(config.path);
  }

  if (e.detail?.verb) {
    return e.detail.verb === method && pattern.test(e.detail.path);
  }

  return false;
}

// Check the status of an HTMX request.
export function checkStatus(e, status) {
  return e.detail?.xhr?.status === status;
}