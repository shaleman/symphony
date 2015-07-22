// node.js
// Display node level info

var AltaPanel = require("./altaView")

// Node Panel
var NodePanel = React.createClass({
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
          return (
                <AltaPanel alta={alta} />
          );
      });

      if (altaListItems.length === 0) {
          altaListItems = <div> No Containers </div>
      }

      var hdr = this.props.nodeInfo.HostName + "       (" + this.props.nodeInfo.HostAddr + ")"

      // Render the DOM elements
      return (
        <Panel header={hdr} bsStyle={titleColor} style={panelStyle}>
            {altaListItems}
        </Panel>
    );
  }
});

module.exports = NodePanel
