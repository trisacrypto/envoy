/*
Allows us to associate instances with elements in the DOM similar to the Boostrap
elementMap model of data storage.

See: https://github.com/twbs/bootstrap/blob/main/js/src/dom/data.js
*/

// Maps elements to instances.
const elementMap = new Map();

export default {
  set(element, key, instance) {
    if (!elementMap.has(element)) {
      elementMap.set(element, new Map());
    }

    const instanceMap = elementMap.get(element);

    // we only want one instance per element
    if (!instanceMap.has(key) && instanceMap.size !== 0) {
      throw new Error(`Cannot bind more than one instance per element. Bound instance: ${Array.from(instanceMap.keys())[0]}.`);
    }

    instanceMap.set(key, instance);
  },

  get(element, key) {
    if (elementMap.has(element)) {
      return elementMap.get(element).get(key) || null;
    }
    return null;
  },

  remove(element, key) {
    if (!elementMap.has(element)) {
      return;
    }

    const instanceMap = elementMap.get(element);
    instanceMap.delete(key);

    // Free up element references if there are no more instances
    if (instanceMap.size === 0) {
      elementMap.delete(element);
    }
  }
}