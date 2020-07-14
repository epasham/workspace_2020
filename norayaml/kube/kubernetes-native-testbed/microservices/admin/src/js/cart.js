const {ShowRequest, ShowResponse, AddRequest, RemoveRequest, CommitRequest, CommitResponse, Cart} = require('./protobuf/cart_pb.js');

const {CartAPIClient, CartAPIPromiseClient} = require('./protobuf/cart_grpc_web_pb.js');

const {GetTokenMetadata} = require('./cookie.js');

export const order = new Vue({
  el: '#cart',
  data: {
    endpoint: window.location.protocol + '//' + window.location.host + "/cart",
    form: {
      userUUID: '',
      cartProducts: [],
    },
    commitform: {
      userUUID: '',
      paymentInfoUUID: '',
      addressUUID: '',
    },
    resp: {
      cart: [],
      errorCode: 0,
      errorMsg: '',
    }
  },
  created: function() {
      this.client = new CartAPIClient(this.endpoint);
      this.promiseClient = new CartAPIPromiseClient(this.endpoint);
  },
  methods: {
    addCartProduct: function() {
      this.form.cartProducts.push({value:''});
    },
    clearForm: function() {
      this.form.userUUID = '';
      this.form.cartProducts = [];
    },
    clearCommitForm: function() {
      this.commitform.userUUID = '';
      this.commitform.paymentInfoUUID = '';
      this.commitform.addressUUID = '';
    },
    clearResponseField: function() {
      this.resp.cart = [];
      this.resp.errorCode = 0;
      this.errorMsg = '';
    },
    showCart: function() {
      this.clearResponseField();
      const req = new ShowRequest();
      req.setUseruuid(this.form.userUUID);
      this.client.show(req, GetTokenMetadata(), (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          let c = new Object();
          c.userUUID = resp.getCart().getUseruuid();
          c.cartProducts = resp.getCart().getCartproductsMap();
          this.resp.cart.push(c);
          this.resp.errorCode = err.code;
        }
      });
    },
    addCart: function() {
      this.clearResponseField();
      const req = new AddRequest();
      const c = new Cart();
      c.setUseruuid(this.form.userUUID);
      this.form.cartProducts.forEach(function(v) {
        c.getCartproductsMap().set(v.productUUID, v.count);
      });
      req.setCart(c);
      this.client.add(req, GetTokenMetadata(), (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          this.resp.errorCode = err.code;
        }
      });
    },
    removeCart: function() {
      this.clearResponseField();
      const req = new RemoveRequest();
      const c = new Cart();
      c.setUseruuid(this.form.userUUID);
      this.form.cartProducts.forEach(function(v) {
        c.getCartproductsMap().set(v.productUUID, v.count);
      });
      req.setCart(c);
      this.client.remove(req, GetTokenMetadata(), (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          this.resp.errorCode = err.code;
        }
      });
    },
    commitCart: async function() {
      this.clearResponseField();
      const req = new CommitRequest();
      const c = new Cart();
      c.setUseruuid(this.commitform.userUUID);

      const sreq = new ShowRequest();
      sreq.setUseruuid(this.commitform.userUUID);
      const resp = await this.promiseClient.show(sreq, GetTokenMetadata());
      const cartProducts = resp.getCart().getCartproductsMap();

      console.log("cartProducts:", cartProducts);
      cartProducts.forEach(function(count, productUUID) {
        console.log("products uuid:", productUUID);
        console.log("products count:", count);
        c.getCartproductsMap().set(productUUID, count);
      });
      req.setCart(c);
      req.setAddressuuid(this.commitform.addressUUID);
      req.setPaymentinfouuid(this.commitform.paymentInfoUUID);
      console.log("cart:", c);
      this.client.commit(req, GetTokenMetadata(), (err, resp) => {
        if (err) {
          this.resp.errorCode = err.code;
          this.resp.errorMsg = err.message;
        } else {
          let c = new Object();
          c.orderUUID = resp.getOrderuuid();
          this.resp.cart.push(c);
          this.resp.errorCode = err.code;
        }
      });
    },
  }
});
