/*
Wraps a container to easily add bootstrap alerts for errors and other messages.
*/

import Data from './data.js';
import { getElement } from './element.js';

class Alerts {
  static get DATA_KEY() {
    return "envoy.Alerts";
  }

  static getInstance(element) {
    return Data.get(getElement(element), this.DATA_KEY)
  }

  constructor(element, config) {
    element = getElement(element);
    if (!element) {
      return;
    }

    const defaultOptions = {
      dismissible: true,
      autoClose: false,
      closeTime: 5000,
      level: 'primary',
      fade: true,
      show: true,
    };

    config = config || {};
    this.config = Object.assign(defaultOptions, config);

    this.container = element;
    Data.set(this.container, this.constructor.DATA_KEY, this);
  }

  dispose() {
    Data.remove(this.container, this.constructor.DATA_KEY);
    for (const propertyName of Object.getOwnPropertyNames(this)) {
      this[propertyName] = null
    }
  }

  alert(title, message, options) {
    options = options || {};
    const config = {
      ...this.config,
      ...options,
    }

    const alert = document.createElement('div');
    alert.setAttribute('role', 'alert');
    alert.classList.add('alert', `alert-${config.level}`);
    if (config.fade) alert.classList.add('fade');
    if (config.show) alert.classList.add('show');

    if (title) {
      alert.innerHTML = `<strong>${title}</strong> ${message}`;
    } else {
      alert.textContent = message;
    }

    if (config.dismissible) {
      alert.classList.add('alert-dismissible');
      alert.insertAdjacentHTML('beforeend', `<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>`);
    }

    this.container.appendChild(alert);

    if (config.autoClose) {
      setTimeout(() => {
        alert.remove();
      }, config.closeTime);
    }

    return alert;
  }

  content(innerHTML, options) {
    options = options || {};
    const config = {
      ...this.config,
      ...options,
    }

    const alert = document.createElement('div');
    alert.setAttribute('role', 'alert');
    alert.classList.add('alert', `alert-${config.level}`);
    if (config.fade) alert.classList.add('fade');
    if (config.show) alert.classList.add('show');

    alert.innerHTML = innerHTML;

    if (config.dismissible) {
      alert.classList.add('alert-dismissible');
      alert.insertAdjacentHTML('beforeend', `<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>`);
    }

    this.container.appendChild(alert);

    if (config.autoClose) {
      setTimeout(() => {
        alert.remove();
      }, config.closeTime);
    }

    return alert;
  }

  primary(title, message, options) {
    return this.alert(title, message, { ...options, level: 'primary' });
  }

  secondary(title, message, options) {
    return this.alert(title, message, { ...options, level: 'secondary' });
  }

  success(title, message, options) {
    return this.alert(title, message, { ...options, level: 'success' });
  }

  danger(title, message, options) {
    console.log('danger');
    return this.alert(title, message, { ...options, level: 'danger' });
  }

  warning(title, message, options) {
    return this.alert(title, message, { ...options, level: 'warning' });
  }

  info(title, message, options) {
    return this.alert(title, message, { ...options, level: 'info' });
  }

  light(title, message, options) {
    return this.alert(title, message, { ...options, level: 'light' });
  }

  dark(title, message, options) {
    return this.alert(title, message, { ...options, level: 'dark' });
  }

}

export default Alerts;