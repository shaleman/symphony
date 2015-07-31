/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};

/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {

/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId])
/******/ 			return installedModules[moduleId].exports;

/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			exports: {},
/******/ 			id: moduleId,
/******/ 			loaded: false
/******/ 		};

/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);

/******/ 		// Flag the module as loaded
/******/ 		module.loaded = true;

/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}


/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;

/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;

/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";

/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(0);
/******/ })
/************************************************************************/
/******/ ([
/* 0 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM *//** @jsx React.DOM */

	// Little hack to make ReactBootstrap components visible globally
	Object.keys(ReactBootstrap).forEach(function (name) {
	    window[name] = ReactBootstrap[name];
	});

	// Navigation tab
	var ControlledTabArea = __webpack_require__(1)

	// Render the main tabs
	React.render(React.createElement(ControlledTabArea, null), document.getElementById('mainViewContainer'));


/***/ },
/* 1 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// navTab.js
	// Navigation tab

	// Node
	var NodePanel = __webpack_require__(2)
	var NetworkPane = __webpack_require__(3)
	var AppPane = __webpack_require__(4)
	var PolicyPane = __webpack_require__(5)
	var VolumesPane = __webpack_require__(6)

	// Define tabs
	var ControlledTabArea = React.createClass({displayName: "ControlledTabArea",
	  getInitialState: function() {
	    return {
	      key: 1,
	      nodes: [],
	      altas: []
	    };
	  },

	  getStateFromServer: function() {
	    // Fetch all nodes
	    $.ajax({
	      url: "/node/",
	      dataType: 'json',
	      success: function(data) {
	        // Sort the data
	        data = data.sort(function(first, second) {
	          if (first.HostAddr > second.HostAddr) {
	              return 1
	          } else if (first.HostAddr < second.HostAddr) {
	              return -1
	          }

	          return 0
	        });

	        this.setState({nodes: data});

	        // FIXME: Save it in a global variable for debug
	        window.globalNodes = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/node/", status, err.toString());
	      }.bind(this)
	    });

	    // Fetch all alta containers
	    $.ajax({
	      url: "/alta/",
	      dataType: 'json',
	      success: function(data) {
	          // Check if we got empty list
	          if (data === null) {
	              this.setState({altas: []});
	              return
	          }
	        // Sort the data
	        data = data.sort(function(first, second) {
	            if (first.Spec.AltaId > second.Spec.AltaId) {
	                return 1
	            } else if (first.Spec.AltaId < second.Spec.AltaId) {
	                return -1
	            }

	            return 0
	        });

	        this.setState({altas: data});

	        // FIXME: Save it in a global variable for debug
	        window.globalAltas = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/alta/", status, err.toString());
	      }.bind(this)
	    });

	    // Sort function for all contiv objects
	    var sortObjFunc = function(first, second) {
	      if (first.key > second.key) {
	          return 1
	      } else if (first.key < second.key) {
	          return -1
	      }

	      return 0
	    }

	    // Get all apps
	    $.ajax({
	      url: "/api/apps/",
	      dataType: 'json',
	      success: function(data) {

	        // Sort the data
	        data = data.sort(sortObjFunc);

	        this.setState({apps: data});

	        // Save it in a global variable for debug
	        window.globalApps = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/api/apps/", status, err.toString());
	      }.bind(this)
	    });

	      // Get all services
	      $.ajax({
	        url: "/api/services/",
	        dataType: 'json',
	        success: function(data) {

	          // Sort the data
	          data = data.sort(sortObjFunc);

	          this.setState({services: data});

	          // Save it in a global variable for debug
	          window.globalServices = data
	        }.bind(this),
	        error: function(xhr, status, err) {
	          console.error("/api/services/", status, err.toString());
	        }.bind(this)
	      });

	    // Get all networks
	    $.ajax({
	      url: "/api/networks/",
	      dataType: 'json',
	      success: function(data) {

	        // Sort the data
	        data = data.sort(sortObjFunc);

	        this.setState({networks: data});

	        // Save it in a global variable for debug
	        window.globalNetworks = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/api/networks/", status, err.toString());
	      }.bind(this)
	    });

	    // Get all endpoint groups
	    $.ajax({
	      url: "/api/endpointGroups/",
	      dataType: 'json',
	      success: function(data) {

	        // Sort the data
	        data = data.sort(sortObjFunc);

	        this.setState({endpointGroups: data});

	        // Save it in a global variable for debug
	        window.globalEndpointGroups = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/api/endpointGroups/", status, err.toString());
	      }.bind(this)
	    });

	    // Get all volumes
	    $.ajax({
	      url: "/api/volumes/",
	      dataType: 'json',
	      success: function(data) {

	        // Sort the data
	        data = data.sort(sortObjFunc);

	        this.setState({volumes: data});

	        // Save it in a global variable for debug
	        window.globalVolumes = data
	      }.bind(this),
	      error: function(xhr, status, err) {
	        console.error("/api/volumes/", status, err.toString());
	      }.bind(this)
	    });
	  },
	  componentDidMount: function() {
	    this.getStateFromServer();

	    // Get state every 2 sec
	    setInterval(this.getStateFromServer, 2000);
	  },
	  handleSelect: function(key) {
	    console.log('selected Tab ' + key);
	    this.setState({key: key});
	  },

	  render: function() {
	      var self = this
	      var nodePanels = this.state.nodes.map(function (node) {
	          return (
	            React.createElement(NodePanel, {key: node.HostAddr, nodeInfo: node, altas: self.state.altas})
	          );
	      });
	    return (
	      React.createElement(TabbedArea, {activeKey: this.state.key, onSelect: this.handleSelect}, 
	        React.createElement(TabPane, {eventKey: 1, tab: "Home"}, " ", React.createElement("h3", null, " Hosts "), 
	             nodePanels 
	        ), 
	        React.createElement(TabPane, {eventKey: 2, tab: "Applications"}, 
	            React.createElement(AppPane, {key: "applications", apps: this.state.apps, services: this.state.services})
	        ), 
	        React.createElement(TabPane, {eventKey: 3, tab: "Networks"}, " ", React.createElement("h3", null, " Networks "), 
	            React.createElement(NetworkPane, {key: "networks", networks: this.state.networks})
	        ), 
	        React.createElement(TabPane, {eventKey: 4, tab: "Policy"}, " ", React.createElement("h3", null, " Policy "), 
	            React.createElement(PolicyPane, {key: "policy", endpointGroups: this.state.endpointGroups})
	        ), 
	        React.createElement(TabPane, {eventKey: 5, tab: "Volumes"}, " ", React.createElement("h3", null, " Volumes "), 
	            React.createElement(VolumesPane, {key: "volumes", volumes: this.state.volumes})
	        )
	      )
	    );
	  }
	});

	module.exports = ControlledTabArea


/***/ },
/* 2 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// nodeView.js
	// Display node level info

	// var AltaPanel = require("./altaView")

	// Node Panel
	var NodePanel = React.createClass({displayName: "NodePanel",
	  render: function() {
	      var self = this

	      // Determine the color of the panel
	      if (this.props.nodeInfo.Fsm.FsmState == "alive") {
	          titleColor = 'success'
	      } else if (this.props.nodeInfo.Fsm.FsmState == "down") {
	          titleColor = 'danger'
	      } else {
	          titleColor = 'warning'
	      }

	      var panelStyle = {
	          width: '90%',
	          minWidth: '350px',
	          float: 'left',
	          margin: '1%',
	      }
	      // Walk thru all the altas and see which ones are on this node
	      var altaListItems = this.props.altas.filter(function(alta) {
	          if (alta.CurrNode === self.props.nodeInfo.HostAddr) {
	              return true
	          }
	          return false
	      }).map(function(alta){
	          if (alta.Spec.Endpoints != null) {
	              var ipAddrList = alta.Spec.Endpoints.map(function(netif) {
	                  return (React.createElement("p", null, " ", netif.NetworkName, " : ", netif.IntfIpv4Addr, " "))
	              });
	          } else {
	              var ipAddrList = "None"
	          }
	          if (alta.Spec.Volumes != null) {
	              var volumeList = alta.Spec.Volumes.map(function(volume) {
	                  return (React.createElement("p", null, " ", volume.BindMountPoint, " : ", volume.DatastoreVolumeId, " "))
	              })
	          } else {
	              var volumeList = "None"
	          }

	          return (
	              React.createElement("tr", {key: alta.Spec.AltaId, className: "info"}, 
	                React.createElement("td", null, alta.Spec.AltaName), 
	                React.createElement("td", null, alta.Spec.Image), 
	                React.createElement("td", null, alta.FsmState), 
	                React.createElement("td", null, ipAddrList), 
	                React.createElement("td", null, volumeList)
	              )
	          );
	      });

	      var altaListView
	      if (altaListItems.length === 0) {
	          altaListView = React.createElement("div", null, " No Containers ")
	      } else {
	          altaListView = React.createElement(Table, {hover: true}, 
	              React.createElement("thead", null, 
	                React.createElement("tr", null, 
	                  React.createElement("th", null, "Container Name"), 
	                  React.createElement("th", null, "Image"), 
	                  React.createElement("th", null, "Status"), 
	                  React.createElement("th", null, "IP Address"), 
	                  React.createElement("th", null, "Volume")
	                )
	              ), 
	              React.createElement("tbody", null, 
	                  altaListItems
	              )
	          )
	      }

	      var hdr = this.props.nodeInfo.HostName + "       (" + this.props.nodeInfo.HostAddr + ")"

	      // Render the DOM elements
	      return (
	        React.createElement(Panel, {header: hdr, bsStyle: titleColor, style: panelStyle}, 
	            altaListView
	        )
	    );
	  }
	});

	module.exports = NodePanel


/***/ },
/* 3 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// network.js
	// Display Network information

	var NetworkPane = React.createClass({displayName: "NetworkPane",
	  	render: function() {
			var self = this

			if (self.props.networks === undefined) {
				return React.createElement("div", null, " ")
			}

			// Walk thru all the altas and see which ones are on this node
			var netListView = self.props.networks.map(function(network){
				if (network.isPublic) {
					netType = "public"
				} else {
					netType = "private"
				}
				return (
					React.createElement("tr", {key: network.key, className: "info"}, 
						React.createElement("td", null, network.tenantName), 
						React.createElement("td", null, network.networkName), 
						React.createElement("td", null, netType), 
						React.createElement("td", null, network.encap), 
						React.createElement("td", null, network.subnet)
					)
				);
			});

			// Render the pane
			return (
	        React.createElement("div", {style: {margin: '5%',}}, 
				React.createElement(Table, {hover: true}, 
					React.createElement("thead", null, 
						React.createElement("tr", null, 
							React.createElement("th", null, "Tenant"), 
							React.createElement("th", null, "Network"), 
							React.createElement("th", null, "Type"), 
							React.createElement("th", null, "Encapsulation"), 
							React.createElement("th", null, "Subnet")
						)
					), 
					React.createElement("tbody", null, 
	            		netListView
					)
				)
	        )
	    );
		}
	});

	module.exports = NetworkPane


/***/ },
/* 4 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// appView.js
	// Render Application tab

	const NewAppModal = React.createClass({displayName: "NewAppModal",
		onSave: function(e) {
			var appName = this.refs.appName.getValue();
			var appParams = {
				tenantName: 'default',
				appName: appName
			};
			console.log("Saving application " + JSON.stringify(appParams))

			$.ajax({
		      url: '/api/apps/default:' + appName + '/',
		      dataType: 'json',
		      type: 'POST',
		      data: JSON.stringify(appParams),
		      success: function(data) {
				console.log("Successfully saved the app " + JSON.stringify(appParams))
		      }.bind(this),
		      error: function(xhr, status, err) {
		        console.error('/api/apps/default:' + appName + '/', status, err.toString());
		      }.bind(this)
		    });

			// close the modal
			this.props.onRequestHide(e)
		},
	  	render:function() {
		    return (
		      React.createElement(Modal, React.__spread({},  this.props, {bsStyle: "primary", title: "New Application", animation: false}), 
		        React.createElement("div", {className: "modal-body"}, 
				React.createElement(Input, {type: "text", label: "Application Name", ref: "appName", placeholder: "Enter name"})
				), 
		        React.createElement("div", {className: "modal-footer"}, 
					React.createElement(Button, {onClick: this.onSave}, "Save"), 
					React.createElement(Button, {onClick: this.props.onRequestHide}, "Close")
		        )
		      )
		    );
	  	}
	});

	const NewServiceModal = React.createClass({displayName: "NewServiceModal",
		onSave: function(e) {
			var serviceName = this.refs.serviceName.getValue();
			if (this.refs.networks.getValue() === "") {
				var networks = []
			} else {
				var networks = this.refs.networks.getValue().split(",").map(function(str){return str.trim()});
			}

			if (this.refs.endpointGroups.getValue() === "") {
				var endpointGroups = [];
			} else {
				var endpointGroups = this.refs.endpointGroups.getValue().split(",").map(function(str){return str.trim()});
			}

			if (this.refs.environment.getValue() === "") {
				var environment = [];
			} else {
				var environment = this.refs.environment.getValue().split(",").map(function(str){return str.trim()});
			}

			var srvParams = {
				tenantName: this.props.tenantName,
				appName: this.props.appName,
				serviceName: serviceName,
				imageName: this.refs.imageName.getValue(),
				command: this.refs.command.getValue(),
				cpu: this.refs.cpu.getValue(),
				memory: this.refs.memory.getValue(),
				networks: networks,
				volumeProfile: this.refs.volumeProfile.getValue(),
				endpointGroups: endpointGroups,
				environment: environment,
				scale: parseInt(this.refs.scale.getValue()),
			};
			console.log("Saving service " + JSON.stringify(srvParams))

			$.ajax({
		      url: '/api/services/default:' + this.props.appName + ':' + serviceName + '/',
		      dataType: 'json',
		      type: 'POST',
		      data: JSON.stringify(srvParams),
		      success: function(data) {
				console.log("Successfully saved the app " + JSON.stringify(srvParams))
		      }.bind(this),
		      error: function(xhr, status, err) {
		        console.error('/api/services/default:' + this.props.appName + ':' + serviceName + '/', status, err.toString());
		      }.bind(this)
		    });

			// close the modal
			this.props.onRequestHide(e)
		},
	  	render:function() {
		    return (
		      React.createElement(Modal, React.__spread({},  this.props, {bsStyle: "primary", bsSize: "large", title: "New Service", animation: false}), 
		        React.createElement("div", {className: "modal-body", style: {margin: '5%',}}, 
					React.createElement(Input, {type: "text", label: "Service Name", ref: "serviceName", placeholder: "Enter service name"}), 
					React.createElement(Input, {type: "text", label: "Image Name", ref: "imageName", placeholder: "Enter image name"}), 
					React.createElement(Input, {type: "text", label: "Command", ref: "command", placeholder: "Enter command"}), 
					React.createElement(Input, {type: "text", label: "Cpus", ref: "cpu", placeholder: "Cpus"}), 
					React.createElement(Input, {type: "text", label: "Memory", ref: "memory", placeholder: "Memory"}), 
					React.createElement(Input, {type: "text", label: "Networks", ref: "networks", placeholder: "Enter networks"}), 
					React.createElement(Input, {type: "text", label: "Volume Profile", ref: "volumeProfile", placeholder: "Enter volume profile name"}), 
					React.createElement(Input, {type: "text", label: "Endpoint Groups", ref: "endpointGroups", placeholder: "Enter endpoint groups"}), 
					React.createElement(Input, {type: "text", label: "Environment Variables", ref: "environment", placeholder: "Enter environment variables"}), 
					React.createElement(Input, {type: "text", label: "Scale", ref: "scale", placeholder: "Enter scale"})
				), 
		        React.createElement("div", {className: "modal-footer"}, 
					React.createElement(Button, {onClick: this.onSave}, "Save"), 
					React.createElement(Button, {onClick: this.props.onRequestHide}, "Close")
		        )
		      )
		    );
	  	}
	});

	const ServiceInfoModal = React.createClass({displayName: "ServiceInfoModal",
		onSave: function(e) {
			var serviceName = this.refs.serviceName.getValue();
			if (this.refs.networks.getValue() === "") {
				var networks = []
			} else {
				var networks = this.refs.networks.getValue().split(",").map(function(str){return str.trim()});
			}

			if (this.refs.endpointGroups.getValue() === "") {
				var endpointGroups = [];
			} else {
				var endpointGroups = this.refs.endpointGroups.getValue().split(",").map(function(str){return str.trim()});
			}

			if (this.refs.environment.getValue() === "") {
				var environment = [];
			} else {
				var environment = this.refs.environment.getValue().split(",").map(function(str){return str.trim()});
			}

			var srvParams = {
				tenantName: this.props.tenantName,
				appName: this.props.appName,
				serviceName: serviceName,
				imageName: this.refs.imageName.getValue(),
				command: this.refs.command.getValue(),
				cpu: this.refs.cpu.getValue(),
				memory: this.refs.memory.getValue(),
				networks: networks,
				endpointGroups: endpointGroups,
				environment: environment,
				scale: parseInt(this.refs.scale.getValue()),
			};
			console.log("Saving service " + JSON.stringify(srvParams))

			$.ajax({
			url: '/api/services/default:' + this.props.appName + ':' + serviceName + '/',
			dataType: 'json',
			type: 'POST',
			data: JSON.stringify(srvParams),
			success: function(data) {
				console.log("Successfully saved the app " + JSON.stringify(srvParams))
			}.bind(this),
			error: function(xhr, status, err) {
				console.error('/api/services/default:' + this.props.appName + ':' + serviceName + '/', status, err.toString());
			}.bind(this)
			});

			// close the modal
			this.props.onRequestHide(e)
		},
	  	render:function() {
			var srv = this.props.service
		    return (
		      React.createElement(Modal, React.__spread({},  this.props, {bsStyle: "primary", bsSize: "large", title: "New Service", animation: false}), 
		        React.createElement("div", {className: "modal-body", style: {margin: '5%',}}, 
					React.createElement(Input, {type: "text", label: "Service Name", ref: "serviceName", defaultValue: srv.serviceName, placeholder: "Enter service name"}), 
					React.createElement(Input, {type: "text", label: "Image Name", ref: "imageName", defaultValue: srv.imageName, placeholder: "Enter image name"}), 
					React.createElement(Input, {type: "text", label: "Command", ref: "command", defaultValue: srv.command, placeholder: "Enter command"}), 
					React.createElement(Input, {type: "text", label: "Cpus", ref: "cpu", defaultValue: srv.cpu, placeholder: "Cpus"}), 
					React.createElement(Input, {type: "text", label: "Memory", ref: "memory", defaultValue: srv.memory, placeholder: "Memory"}), 
					React.createElement(Input, {type: "text", label: "Networks", ref: "networks", defaultValue: srv.networks, placeholder: "Enter networks"}), 
					React.createElement(Input, {type: "text", label: "Volume Profile", ref: "volumeProfile", placeholder: "Enter volume profile name"}), 
					React.createElement(Input, {type: "text", label: "Endpoint Groups", ref: "endpointGroups", defaultValue: srv.endpointGroups, placeholder: "Enter endpoint groups"}), 
					React.createElement(Input, {type: "text", label: "Environment Variables", ref: "environment", defaultValue: srv.environment, placeholder: "Enter environment variables"}), 
					React.createElement(Input, {type: "text", label: "Scale", ref: "scale", defaultValue: srv.scale, placeholder: "Enter scale"})
				), 
		        React.createElement("div", {className: "modal-footer"}, 
					React.createElement(Button, {onClick: this.onSave}, "Save"), 
					React.createElement(Button, {onClick: this.props.onRequestHide}, "Close")
		        )
		      )
		    );
	  	}
	});

	var ServiceSummary = React.createClass({displayName: "ServiceSummary",
		handleServiceClick: function() {
			console.log("Clicked on service %s", this.props.service.serviceName)
		},
	  	render: function() {
			var self = this
			var srv = self.props.service

			// List the networks
			if (srv.networks !== undefined) {
				var networks = srv.networks.reduce(function(a, b){
					return a + ", " + b;
				});
			} else {
				var networks = "None"
			}

			// Render the row
			return (
				React.createElement(ModalTrigger, {modal: React.createElement(ServiceInfoModal, {tenantName: "default", appName: self.props.app.appName, service: srv})}, 
					React.createElement("tr", {className: "info"}, 
						React.createElement("td", null, srv.serviceName), 
						React.createElement("td", null, srv.imageName), 
						React.createElement("td", null, networks), 
						React.createElement("td", null, srv.command), 
						React.createElement("td", null, srv.scale)
					)
				)
			);
		}
	});

	var AppPane = React.createClass({displayName: "AppPane",
		handleNewAppClick: function() {
			console.log("New applicaiton clicked")
			this.setState({ showModal: true });
			// $(this.refs.payload.getDOMNode()).modal();
		},
		handleNewServiceClick: function() {
			console.log("New service clicked")
		},
		getInitialState:function(){
	    	return { showModal: false };
	  	},
		closeModal: function() {
			log.console("Closing modal")
			this.setState({ showModal: false });
		},
	  	render: function() {
			var self = this

			if ((self.props.apps === undefined) ||(self.props.services === undefined)) {
				return React.createElement("div", null)
			}

			// Walk all apps and display it
			appListView = self.props.apps.map(function(app){
				// walk all services and filter the ones for this app
				var serviceListView = self.props.services.filter(function(srv) {
					if ((srv.tenantName === app.tenantName) && (srv.appName === app.appName)) {
						return true
					}
					return false
				}).map(function(srv){
					return (
						React.createElement(ServiceSummary, {key: srv.key, app: app, service: srv})
					);
				});

				var hdr = React.createElement("h4", null, " ", app.appName, " ")

				return (
					React.createElement(Panel, {key: app.key, header: hdr, bsStyle: "success"}, 
						React.createElement(Grid, {fluid: true}, 
							React.createElement(Row, null, 
								React.createElement(Col, {xs: 3, md: 2}, " ", React.createElement("h3", null, " Services"), " "), 
								React.createElement(Col, {xs: 3, md: 2}, 
									React.createElement(ModalTrigger, {modal: React.createElement(NewServiceModal, {tenantName: "default", appName: app.appName})}, 
										React.createElement(OverlayTrigger, {placement: "right", overlay: React.createElement(Tooltip, null, "Add new service  ")}, 
											React.createElement(Button, {bsStyle: "info", style: {margin: '5%',}}, React.createElement(Glyphicon, {glyph: "plus"}))
										)
									)
								)
							)
						), 
						React.createElement(Table, {hover: true}, 
							React.createElement("thead", null, 
								React.createElement("tr", null, 
									React.createElement("th", null, "Service Name"), 
									React.createElement("th", null, "Image Name"), 
									React.createElement("th", null, "Networks"), 
									React.createElement("th", null, "Command"), 
									React.createElement("th", null, "Scale")
								)
							), 
							React.createElement("tbody", null, 
				            	serviceListView
							)
						)
			        )
				);
			});

			// Render the Application tab
			return (
				React.createElement("div", null, 
				React.createElement(Grid, {fluid: true}, 
					React.createElement(Row, null, 
						React.createElement(Col, {xs: 3, md: 2}, " ", React.createElement("h3", null, " Applications"), " "), 
						React.createElement(Col, {xs: 3, md: 2}, " ", React.createElement(ModalTrigger, {modal: React.createElement(NewAppModal, null)}, 
							React.createElement(OverlayTrigger, {placement: "right", overlay: React.createElement(Tooltip, null, "Add new application  ")}, 
						    	React.createElement(Button, {bsSize: "large", style: {margin: '5%',}}, React.createElement(Glyphicon, {glyph: "plus"}))
							)
						))
					)
				), 
				React.createElement("div", {style: {margin: '1%',}}, 
					appListView
				), 
				React.createElement("div", null

				)
				)
			);
		}
	});

	module.exports = AppPane


/***/ },
/* 5 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// policy.js
	// Display Policy information

	var PolicyPane = React.createClass({displayName: "PolicyPane",
	  	render: function() {
			var self = this

			if (self.props.endpointGroups === undefined) {
				return React.createElement("div", null, " ")
			}

			// Walk thru all the altas and see which ones are on this node
			var epgListView = self.props.endpointGroups.map(function(epg){
				return (
					React.createElement("tr", {key: epg.key, className: "info"}, 
						React.createElement("td", null, epg.tenantName), 
						React.createElement("td", null, epg.networkName), 
						React.createElement("td", null, epg.groupName), 
						React.createElement("td", null, epg.policies)
					)
				);
			});

			// Render the pane
			return (
	        React.createElement("div", {style: {margin: '5%',}}, 
				React.createElement(Table, {hover: true}, 
					React.createElement("thead", null, 
						React.createElement("tr", null, 
							React.createElement("th", null, "Tenant"), 
							React.createElement("th", null, "Network"), 
							React.createElement("th", null, "Endpoint Group"), 
							React.createElement("th", null, "Policies")
						)
					), 
					React.createElement("tbody", null, 
	            		epgListView
					)
				)
	        )
	    );
		}
	});

	module.exports = PolicyPane


/***/ },
/* 6 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// volumes.js
	// Display Volumes information

	var VolumesPane = React.createClass({displayName: "VolumesPane",
	  	render: function() {
			var self = this

			if (self.props.volumes === undefined) {
				return React.createElement("div", null, " ")
			}

			// Walk thru all the volumes
			var volListView = self.props.volumes.map(function(vol){
				return (
					React.createElement("tr", {key: vol.key, className: "info"}, 
						React.createElement("td", null, vol.tenantName), 
						React.createElement("td", null, vol.volumeName), 
						React.createElement("td", null, vol.poolName), 
						React.createElement("td", null, vol.size)
					)
				);
			});

			// Render the pane
			return (
	        React.createElement("div", {style: {margin: '5%',}}, 
				React.createElement(Table, {hover: true}, 
					React.createElement("thead", null, 
						React.createElement("tr", null, 
							React.createElement("th", null, "Tenant"), 
							React.createElement("th", null, "Volume"), 
							React.createElement("th", null, "Pool"), 
							React.createElement("th", null, "Size")
						)
					), 
					React.createElement("tbody", null, 
	            		volListView
					)
				)
	        )
	    );
		}
	});

	module.exports = VolumesPane


/***/ }
/******/ ]);
