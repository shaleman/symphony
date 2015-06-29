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
	  },
	  componentDidMount: function() {
	    this.getStateFromServer();
	    // FIXME: Uncomment this to get state every 2 sec
	    setInterval(this.getStateFromServer, 2000);
	  },
	  handleSelect: function(key) {
	    console.log('selected Tab ' + key);
	    this.setState({key: key});
	  },

	  render: function() {
	      var self = this
	      console.log("Rendering tabs")

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
	        React.createElement(TabPane, {eventKey: 2, tab: "Containers"}, " ", React.createElement("h3", null, " Containers"), " "), 
	        React.createElement(TabPane, {eventKey: 3, tab: "Hosts"}, " ", React.createElement("h3", null, " Hosts "), " "), 
	        React.createElement(TabPane, {eventKey: 4, tab: "Volumes"}, " ", React.createElement("h3", null, " Volumes "), " "), 
	        React.createElement(TabPane, {eventKey: 5, tab: "Networks"}, " ", React.createElement("h3", null, " Networks "), " ")
	      )
	    );
	  }
	});

	module.exports = ControlledTabArea


/***/ },
/* 2 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// node.js
	// Display node level info

	var AltaPanel = __webpack_require__(3)

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
	          width: '30%',
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
	          return (
	                React.createElement(AltaPanel, {alta: alta})
	          );
	      });

	      if (altaListItems.length === 0) {
	          altaListItems = React.createElement("div", null, " No Containers ")
	      }

	      var hdr = this.props.nodeInfo.HostName + "       (" + this.props.nodeInfo.HostAddr + ")"

	      // Render the DOM elements
	      return (
	        React.createElement(Panel, {header: hdr, bsStyle: titleColor, style: panelStyle}, 
	            altaListItems
	        )
	    );
	  }
	});

	module.exports = NodePanel


/***/ },
/* 3 */
/***/ function(module, exports, __webpack_require__) {

	/** @jsx React.DOM */// alta.js
	// Display Alta container info

	var AltaPanel = React.createClass({displayName: "AltaPanel",
	  render: function() {
	      var self = this

	      // Determine the color of the panel
	      if (this.props.alta.FsmState == "running") {
	          titleColor = 'success'
	      } else if (this.props.alta.FsmState == "failed") {
	          titleColor = 'danger'
	      } else {
	          titleColor = 'warning'
	      }

	      var memory = this.props.alta.Spec.Memory / ( 1024 * 1024)

	      // display all attached volumes
	      var volumes = this.props.alta.Spec.Volumes.map(function(volume){
	          return (
	              React.createElement(ListGroupItem, {key: volume.BindMountPoint}, 
	                React.createElement("div", null, " ", React.createElement("h4", null, " ", volume.BindMountPoint, " "), " "), 
	                React.createElement("div", null, " ", volume.DatastoreType, " : ", volume.DatastoreVolumeId, " ")
	              )
	          );
	      });

	      // Display all attached network intf
	      var netIfs = this.props.alta.Spec.NetworkIfs.map(function(netif){
	          return (
	              React.createElement(ListGroupItem, {key: netif.IntfIpv4Addr}, 
	                React.createElement("div", null, " ", React.createElement("h4", null, " ", netif.IntfIpv4Addr, " "), " "), 
	                React.createElement("div", null, " Network: ", netif.NetworkName, " "), 
	                React.createElement("div", null, " Mac Addr: ", netif.IntfMacAddr, " ")
	              )
	          );
	      });

	      var panelHdr = "Container: "
	      if (this.props.alta.Spec.AltaName !== "") {
	          panelHdr = panelHdr + this.props.alta.Spec.AltaName
	      } else {
	          panelHdr = panelHdr + this.props.alta.Spec.AltaId
	      }
	      return (
	          React.createElement(Panel, {header: panelHdr, bsStyle: "primary"}, 
	            React.createElement("h4", null, " ", React.createElement(Label, {bsStyle: titleColor, style: {color: 'black', shadow: 'none'}}, 
	                this.props.alta.FsmState
	            )), 
	            React.createElement("div", null, " ", React.createElement("h4", null, " Image: ", this.props.alta.Spec.Image, " "), " "), 
	            React.createElement("div", null, " Name: ", this.props.alta.Spec.AltaName, " ", React.createElement("br", null), " Id: ", this.props.alta.Spec.AltaId), 
	            React.createElement("div", null, " Cpu: ", this.props.alta.Spec.NumCpu, " "), 
	            React.createElement("div", null, " Memory: ", memory, " MB "), 
	            React.createElement(ListGroup, null, 
	                React.createElement(ListGroupItem, {bsStyle: "info"}, "Volumes"), 
	                volumes
	            ), 
	            React.createElement(ListGroup, null, 
	                React.createElement(ListGroupItem, {bsStyle: "info"}, "Network interfaces"), 
	                netIfs
	            )
	          )
	      );
	  }
	});

	// Export the panel
	module.exports = AltaPanel


/***/ }
/******/ ]);
