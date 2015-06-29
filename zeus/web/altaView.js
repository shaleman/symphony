// alta.js
// Display Alta container info

var AltaPanel = React.createClass({
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
              <ListGroupItem key={volume.BindMountPoint}>
                <div> <h4> {volume.BindMountPoint} </h4> </div>
                <div> {volume.DatastoreType} : {volume.DatastoreVolumeId} </div>
              </ListGroupItem>
          );
      });

      // Display all attached network intf
      var netIfs = this.props.alta.Spec.NetworkIfs.map(function(netif){
          return (
              <ListGroupItem key={netif.IntfIpv4Addr}>
                <div> <h4> {netif.IntfIpv4Addr} </h4> </div>
                <div> Network: {netif.NetworkName} </div>
                <div> Mac Addr: {netif.IntfMacAddr} </div>
              </ListGroupItem>
          );
      });

      var panelHdr = "Container: "
      if (this.props.alta.Spec.AltaName !== "") {
          panelHdr = panelHdr + this.props.alta.Spec.AltaName
      } else {
          panelHdr = panelHdr + this.props.alta.Spec.AltaId
      }
      return (
          <Panel header={panelHdr} bsStyle='primary'>
            <h4> <Label bsStyle={titleColor} style={{color: 'black', shadow: 'none'}}>
                {this.props.alta.FsmState}
            </Label></h4>
            <div> <h4> Image: {this.props.alta.Spec.Image} </h4> </div>
            <div> Name: {this.props.alta.Spec.AltaName} <br/> Id: {this.props.alta.Spec.AltaId}</div>
            <div> Cpu: {this.props.alta.Spec.NumCpu} </div>
            <div> Memory: {memory} MB </div>
            <ListGroup>
                <ListGroupItem bsStyle='info'>Volumes</ListGroupItem>
                {volumes}
            </ListGroup>
            <ListGroup>
                <ListGroupItem bsStyle='info'>Network interfaces</ListGroupItem>
                {netIfs}
            </ListGroup>
          </Panel>
      );
  }
});

// Export the panel
module.exports = AltaPanel
