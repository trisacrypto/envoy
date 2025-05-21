/*
Checks the htmx requestConfig from the event detail (if any) and determines if the
path and method match what the request was for; allowing you to disambiguate between
different htmx requests on the same page.
*/
export function isRequestFor(e, path, method) {
  method = method.toLowerCase();

  // Check the request config for the path and method if it has been configured.
  const config = e.detail?.requestConfig;
  if (config) {
    return urlPath(config.path) === path && config.verb === method;
  }

  // Check the detail directly if this is during a request config event.
  if (e.detail?.path && e.detail?.verb) {
    return urlPath(e.detail.path) === path && e.detail.verb === method;
  }

  // Otherwise return false since we can't determine the configuration.
  return false;
}

/*
Checks the htmx requestConfig from the event detail (if any) and determines if the
method match what the request was for; allowing you to disambiguate between different
htmx requests with different HTTP verbs on the same page.
*/
export function isRequestMethod(e, method) {
  method = method.toLowerCase();

  // Check the request config for the path and method if it has been configured.
  const config = e.detail?.requestConfig;
  if (config) {
    return config.verb === method;
  }

  // Check the detail directly if this is during a request config event.
  if (e.detail?.verb) {
    return e.detail.verb === method;
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
  method = method.toLowerCase();

  if (typeof(pattern) === 'string') {
    pattern = new RegExp(pattern);
  }

  if (!pattern instanceof RegExp) {
    throw new Error('request pattern for the path must be a string or RegExp');
  }

  const config = e.detail?.requestConfig;
  if (config) {
    return config.verb === method && pattern.test(urlPath(config.path));
  }

  if (e.detail?.verb) {
    return e.detail.verb === method && pattern.test(urlPath(e.detail.path));
  }

  return false;
}

// Check the status of an HTMX request.
export function checkStatus(e, status) {
  return e.detail?.xhr?.status === status;
}

// Removes any query strings from the path.
export function urlPath(uri) {
  try {
    const url = new URL(uri);
    return url.pathname;
  } catch (InvalidURL) {
    return uri.split('?')[0];
  }
}

// Extracts the query string from the path.
export function urlQuery(uri) {
  try {
    const url = new URL(uri);
    return new URLSearchParams(url.search);
  } catch (InvalidURL) {
    const parts = uri.split('?');
    if (parts.length > 1) {
      return new URLSearchParams(parts[1]);
    } else {
      return new URLSearchParams();
    }
  }
}