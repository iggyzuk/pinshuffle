// Shuffle all packery elements!
Packery.prototype.shuffle = function(){
    var m = this.items.length, t, i;
    while (m) {
        i = Math.floor(Math.random() * m--);
        t = this.items[m];
        this.items[m] = this.items[i];
        this.items[i] = t;
    }
    this.layout();
  }