// nodeView.js
// Display node level info

// var AltaPanel = require("./altaView")

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
          if (alta.Spec.Endpoints != null) {
              var ipAddrList = alta.Spec.Endpoints.map(function(netif) {
                  return (<p> {netif.NetworkName} : {netif.IntfIpv4Addr} </p>)
              });
          } else {
              var ipAddrList = "None"
          }
          if (alta.Spec.Volumes != null) {
              var volumeList = alta.Spec.Volumes.map(function(volume) {
                  return (<p> {volume.BindMountPoint} : {volume.DatastoreVolumeId} </p>)
              })
          } else {
              var volumeList = "None"
          }

          return (
              <tr key={alta.Spec.AltaId} className="info">
                <td>{alta.Spec.AltaName}</td>
                <td>{alta.Spec.Image}</td>
                <td>{alta.FsmState}</td>
                <td>{ipAddrList}</td>
                <td>{volumeList}</td>
              </tr>
          );
      });

      var altaListView
      if (altaListItems.length === 0) {
          altaListView = <div> No Containers </div>
      } else {
          altaListView = <Table hover>
              <thead>
                <tr>
                  <th>Container Name</th>
                  <th>Image</th>
                  <th>Status</th>
                  <th>IP Address</th>
                  <th>Volume</th>
                </tr>
              </thead>
              <tbody>
                  {altaListItems}
              </tbody>
          </Table>
      }

      var hdr = this.props.nodeInfo.HostName + "       (" + this.props.nodeInfo.HostAddr + ")"

      // Render the DOM elements
      return (
        <Panel header={hdr} bsStyle={titleColor} style={panelStyle}>
            {altaListView}
        </Panel>
    );
  }
});

module.exports = NodePanel
