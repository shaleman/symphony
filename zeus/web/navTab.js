// navTab.js
// Navigation tab

// Node
var NodePanel = require("./nodeView")
var NetworkPane = require("./network")
var AppPane = require("./appView")
var PolicyPane = require("./policy")
var VolumesPane = require("./volumes")

// Define tabs
var ControlledTabArea = React.createClass({
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
            <NodePanel key={node.HostAddr} nodeInfo={node} altas={self.state.altas} />
          );
      });
    return (
      <TabbedArea activeKey={this.state.key} onSelect={this.handleSelect}>
        <TabPane eventKey={1} tab='Home'> <h3> Hosts </h3>
            { nodePanels }
        </TabPane>
        <TabPane eventKey={2} tab='Applications'>
            <AppPane key="applications" apps={this.state.apps} services={this.state.services} />
        </TabPane>
        <TabPane eventKey={3} tab='Networks'> <h3> Networks </h3>
            <NetworkPane key="networks" networks={this.state.networks} />
        </TabPane>
        <TabPane eventKey={4} tab='Policy'> <h3> Policy </h3>
            <PolicyPane key="policy" endpointGroups={this.state.endpointGroups} />
        </TabPane>
        <TabPane eventKey={5} tab='Volumes'> <h3> Volumes </h3>
            <VolumesPane key="volumes" volumes={this.state.volumes} />
        </TabPane>
      </TabbedArea>
    );
  }
});

module.exports = ControlledTabArea
