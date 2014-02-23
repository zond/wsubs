
function jsCacheBackend() {
  var that = this;
	var localCache = new Cache(-1, false, new Cache.LocalStorageCacheStorage());
	that.get = function(key) {
		return localCache.getItem(key);
	};
	that.set = function(key, value) {
		return localCache.setItem(key, value);
	};
  return that;
}

function localStorageBackend() {
  var that = this;
	that.get = localStorage.getItem;
	that.set = localStorage.setItem;
	return that;
}

function wsBackbone(opts) {
	var cache_backend = opts.cache_backend;

	var jsock = jSock(opts);

	var subscriptions = {};

	var urlError = function() {
		throw new Error('A "url" property or function must be specified');
	};

	var closeSubscription = function(that) {
		var url = _.result(that, 'url') || urlError(); 
		jsock.unsubscribe(url);
	};

	Backbone.Collection.prototype.close = function() {
		closeSubscription(this);
	};
	Backbone.Model.prototype.close = function() {
		closeSubscription(this);
	};

	Backbone.sync = function(method, model, options) {
		var urlBefore = options.url || _.result(model, 'url') || url_error(); 
		if (method == 'read') {
			var cached = cache_backend.get(urlBefore);
			if (cached != null) {
				jsock.log_debug('Loaded', urlBefore, 'from localStorage');
				var data = JSON.parse(cached);
				jsock.log_trace(data);
				var success = options.success;
				options.success = null;
				success(data, null, options);
			}
		}
		if (method == 'read') {
			jsock.log_debug('Subscribing to', urlBefore);
			jsock.subscribe(urlBefore, function(mobj) {
				jsock.log_debug('Got', mobj.Type, mobj.Object.URI, 'from websocket');
				jsock.log_trace(mobj.Object.Data);
				if (mobj.Type == 'Delete') {
					if (model.models != null) {
						_.each(mobj.Object.Data, function(element) {
							var model = model.get(element.Id);
							model.remove(model, { silent: true });
						});
						model.trigger('reset');
					} else {
						model.trigger('reset');
					}
				} else {
					if (options != null && options.success != null) {
						options.success(mobj.Object.Data, null, options);
						delete(options.success);
					} else {
						model.set(mobj.Object.Data, { remove: mobj.Type == 'Fetch', reset: true });
					}
				}
				if (_.result(model, 'localStorage')) {
					cache_backend.set(mobj.Object.URI, JSON.stringify(model));
					jsock.log_debug('Stored', mobj.Object.URI, 'in localStorage');
				}
			});
		} else if (method == 'create') {
			jsock.log_debug('Creating', urlBefore);
			jsock.state.ws.send_if_ready(JSON.stringify({
				Type: 'Create',
				Object: {
					URI: urlBefore,
					Data: model,
				},
			}));
			if (options.success) {
				var success = options.success;
				options.success = null;
				success(model.toJSON(), null, options);
			}
		} else if (method == 'update') {
			jsock.log_debug('Updating', urlBefore);
			jsock.state.ws.send_if_ready(JSON.stringify({
				Type: 'Update',
				Object: {
					URI: urlBefore,
					Data: model,
				},
			}));
			if (options.success) {
				var success = options.success;
				options.success = null;
				success(model.toJSON(), null, options);
			}
		} else if (method == 'delete') {
			jsock.log_debug('Deleting', urlBefore);
			jsock.state.ws.send_if_ready(JSON.stringify({
				Type: 'Delete',
				Object: {
					URI: urlBefore,
				},
			}));
		} else {
			jsock.log_error("Don't know how to handle " + method);
			if (options.error) {
				options.error(model, "Don't know how to handle " + method, options);
			}
		}
	};
}

