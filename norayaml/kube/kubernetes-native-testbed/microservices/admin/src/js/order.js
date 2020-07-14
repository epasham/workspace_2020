const {GetRequest, GetResponse, SetRequest, SetResponse, UpdateRequest, DeleteRequest, Order, OrderedProduct} = require('./protobuf/order_pb.js');

const {OrderAPIClient} = require('./protobuf/order_grpc_web_pb.js');

export const order = new Vue({
  el: '#order',
  data: {
    endpoint: window.location.protocol + '//' + window.location.host + "/order",
    form: {
      uuid: '',
      userUUID: '',
      paymentInfoUUID: '',
      addressUUID: '',
      orderedProducts: [],
    },
    resp: {
      order: [],
      errorCode: 0,
      errorMsg: '',
    }
  },
  created: function() {
      this.client = new OrderAPIClient(this.endpoint);
  },
  methods: {
    addOrderedProduct: function() {
      this.form.orderedProducts.push({value:''});
    },
    clearForm: function() {
      this.form.uuid = '';
      this.form.userUUID = '';
      this.form.paymentInfoUUID = '';
      this.form.addressUUID = '';
      this.form.orderedProducts = [];
    },
    clearResponseField: function() {
      this.resp.order = [];
      this.resp.errorCode = 0;
      this.errorMsg = '';
    },
    getOrder: function() {
      this.clearResponseField();
      const req = new GetRequest();
      req.setUuid(this.form.uuid);
      this.client.get(req, {}, (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          let o = new Object();
          o.uuid = resp.getOrder().getUuid();
          o.userUUID = resp.getOrder().getUseruuid();
          o.paymentInfoUUID = resp.getOrder().getPaymentinfouuid();
          o.addressUUID = resp.getOrder().getAddressuuid();
          o.orderedProducts = resp.getOrder().getOrderedproductsList();
          o.createdAt = resp.getOrder().getCreatedat();
          o.updatedAt = resp.getOrder().getUpdatedat();
          o.deletedAt = resp.getOrder().getDeletedat();
          this.resp.order.push(o);
          this.resp.errorCode = err.code;
        }
      });
    },
    setOrder: function() {
      this.clearResponseField();
      const req = new SetRequest();
      const o = new Order();
      o.setUseruuid(this.form.userUUID);
      o.setPaymentinfouuid(this.form.paymentInfoUUID);
      o.setAddressuuid(this.form.addressUUID);
      var orderedProducts = []
      this.form.orderedProducts.forEach(function(v) {
        const op = new OrderedProduct();
        op.setProductuuid(v.productUUID);
        op.setCount(v.count);
        op.setPrice(v.price);
        orderedProducts.push(op)
      });
      o.setOrderedproductsList(orderedProducts);
      req.setOrder(o);
      this.client.set(req, {}, (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          let o = new Object();
          o.uuid = resp.getUuid();
          this.resp.order.push(o);
          this.resp.errorCode = err.code;
        }
      });
    },
    updateOrder: function() {
      this.clearResponseField();
      const req = new UpdateRequest();
      const o = new Order();
      o.setUuid(this.form.uuid);
      o.setUseruuid(this.form.userUUID);
      o.setPaymentinfouuid(this.form.paymentInfoUUID);
      o.setAddressuuid(this.form.addressUUID);
      o.setOrderedproductsList(this.form.orderedProducts);
      req.setOrder(o);
      this.client.update(req, {}, (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          this.resp.errorCode = err.code;
        }
      });
    },
    deleteOrder: function() {
      this.clearResponseField();
      const req = new DeleteRequest();
      req.setUuid(this.form.uuid);
      this.client.delete(req, {}, (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          this.resp.errorCode = err.code;
        }
      });
    },
  }
});
