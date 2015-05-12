// navTab.js
// Navigation tab

// Node
var NodePanel = require("./nodeView")

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
            <NodePanel key={node.HostAddr} nodeInfo={node} altas={self.state.altas} />
          );
      });
    return (
      <TabbedArea activeKey={this.state.key} onSelect={this.handleSelect}>
        <TabPane eventKey={1} tab='Home'> <h3> Hosts </h3>
            { nodePanels }
        </TabPane>
        <TabPane eventKey={2} tab='Containers'> <h3> Containers</h3> </TabPane>
        <TabPane eventKey={3} tab='Hosts'> <h3> Hosts </h3> </TabPane>
        <TabPane eventKey={4} tab='Volumes'> <h3> Volumes </h3> </TabPane>
        <TabPane eventKey={5} tab='Networks'> <h3> Networks </h3> </TabPane>
      </TabbedArea>
    );
  }
});

module.exports = ControlledTabArea
