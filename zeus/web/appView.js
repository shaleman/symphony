// appView.js
// Render Application tab

const NewAppModal = React.createClass({
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
	},
  	render() {
	    return (
	      <Modal {...this.props} bsStyle='primary' title='New Application' animation={false}>
	        <div className='modal-body'>
			<Input type='text' label='Application Name' ref='appName' placeholder='Enter name' />
			</div>
	        <div className='modal-footer'>
				<Button onClick={this.onSave}>Save</Button>
				<Button onClick={this.props.onRequestHide}>Close</Button>
	        </div>
	      </Modal>
	    );
  	}
});

const NewServiceModal = React.createClass({
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
	},
  	render() {
	    return (
	      <Modal {...this.props} bsStyle='primary' bsSize='large' title='New Service' animation={false}>
	        <div className='modal-body' style={{margin: '5%',}}>
				<Input type='text' label='Service Name' ref='serviceName' placeholder='Enter service name' />
				<Input type='text' label='Image Name' ref='imageName' placeholder='Enter image name' />
				<Input type='text' label='Command' ref='command' placeholder='Enter command' />
				<Input type='text' label='Cpus' ref='cpu' placeholder='Cpus' />
				<Input type='text' label='Memory' ref='memory' placeholder='Memory' />
				<Input type='text' label='Networks' ref='networks' placeholder='Enter networks' />
				<Input type='text' label='Endpoint Groups' ref='endpointGroups' placeholder='Enter endpoint groups' />
				<Input type='text' label='Environment Variables' ref='environment' placeholder='Enter environment variables' />
				<Input type='text' label='Scale' ref='scale' placeholder='Enter scale' />
			</div>
	        <div className='modal-footer'>
				<Button onClick={this.onSave}>Save</Button>
				<Button onClick={this.props.onRequestHide}>Close</Button>
	        </div>
	      </Modal>
	    );
  	}
});

const ServiceInfoModal = React.createClass({
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
	},
  	render() {
		var srv = this.props.service
	    return (
	      <Modal {...this.props} bsStyle='primary' bsSize='large' title='New Service' animation={false}>
	        <div className='modal-body' style={{margin: '5%',}}>
				<Input type='text' label='Service Name' ref='serviceName' defaultValue={srv.serviceName} placeholder='Enter service name' />
				<Input type='text' label='Image Name' ref='imageName' defaultValue={srv.imageName} placeholder='Enter image name' />
				<Input type='text' label='Command' ref='command' defaultValue={srv.command} placeholder='Enter command' />
				<Input type='text' label='Cpus' ref='cpu' defaultValue={srv.cpu} placeholder='Cpus' />
				<Input type='text' label='Memory' ref='memory' defaultValue={srv.memory} placeholder='Memory' />
				<Input type='text' label='Networks' ref='networks' defaultValue={srv.networks} placeholder='Enter networks' />
				<Input type='text' label='Endpoint Groups' ref='endpointGroups' defaultValue={srv.endpointGroups} placeholder='Enter endpoint groups' />
				<Input type='text' label='Environment Variables' ref='environment' defaultValue={srv.environment} placeholder='Enter environment variables' />
				<Input type='text' label='Scale' ref='scale' defaultValue={srv.scale} placeholder='Enter scale' />
			</div>
	        <div className='modal-footer'>
				<Button onClick={this.onSave}>Save</Button>
				<Button onClick={this.props.onRequestHide}>Close</Button>
	        </div>
	      </Modal>
	    );
  	}
});

var ServiceSummary = React.createClass({
	handleServiceClick: function() {
		console.log("Clicked on service %s", this.props.service.serviceName)
	},
  	render: function() {
		var self = this
		var srv = self.props.service
		var networks = srv.networks.reduce(function(a, b){
			return a + ", " + b;
		});
		return (
			<ModalTrigger modal={<ServiceInfoModal tenantName='default' appName={self.props.app.appName} service={srv}/>}>
				<tr className="info">
					<td>{srv.serviceName}</td>
					<td>{srv.imageName}</td>
					<td>{networks}</td>
					<td>{srv.command}</td>
					<td>{srv.scale}</td>
				</tr>
			</ModalTrigger>
		);
	}
});

var AppPane = React.createClass({
	handleNewAppClick: function() {
		console.log("New applicaiton clicked")
		this.setState({ showModal: true });
		// $(this.refs.payload.getDOMNode()).modal();
	},
	handleNewServiceClick: function() {
		console.log("New service clicked")
	},
	getInitialState(){
    	return { showModal: false };
  	},
	closeModal: function() {
		log.console("Closing modal")
		this.setState({ showModal: false });
	},
  	render: function() {
		var self = this

		if ((self.props.apps === undefined) ||(self.props.services === undefined)) {
			return <div/>
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
					<ServiceSummary key={srv.key} app={app} service={srv} />
				);
			});

			var hdr = <h4> {app.appName} </h4>

			return (
				<Panel key={app.key} header={hdr} bsStyle='success' >
					<Grid fluid={true}>
						<Row>
							<Col xs={3} md={2}> <h3> Services</h3> </Col>
							<Col xs={3} md={2}>
								<ModalTrigger modal={<NewServiceModal tenantName='default' appName={app.appName} />}>
									<OverlayTrigger placement='right' overlay={<Tooltip>Add new service  </Tooltip>}>
										<Button bsStyle='info' style={{margin: '5%',}}><Glyphicon glyph='plus' /></Button>
									</OverlayTrigger>
								</ModalTrigger>
							</Col>
						</Row>
					</Grid>
					<Table hover>
						<thead>
							<tr>
								<th>Service Name</th>
								<th>Image Name</th>
								<th>Networks</th>
								<th>Command</th>
								<th>Scale</th>
							</tr>
						</thead>
						<tbody>
			            	{serviceListView}
						</tbody>
					</Table>
		        </Panel>
			);
		});

		// Render the Application tab
		return (
			<div>
			<Grid fluid={true}>
				<Row>
					<Col xs={3} md={2}> <h3> Applications</h3> </Col>
					<Col xs={3} md={2}> <ModalTrigger modal={<NewAppModal />}>
						<OverlayTrigger placement='right' overlay={<Tooltip>Add new application  </Tooltip>}>
					    	<Button bsSize='large' style={{margin: '5%',}}><Glyphicon glyph='plus' /></Button>
						</OverlayTrigger>
					</ModalTrigger></Col>
				</Row>
			</Grid>
			<div style={{margin: '1%',}}>
				{appListView}
			</div>
			<div>

			</div>
			</div>
		);
	}
});

module.exports = AppPane
