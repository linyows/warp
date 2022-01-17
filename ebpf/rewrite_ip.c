#include "bpf_helpers.h"

#define TARGET_PORT 25
#define REWRITE_IP 3232248325

// Ethernet header
struct ethhdr {
  __u8 h_dest[6];
  __u8 h_source[6];
  __u16 h_proto;
} __attribute__((packed));

// IPv4 header
struct iphdr {
  __u8 ihl : 4;
  __u8 version : 4;
  __u8 tos;
  __u16 tot_len;
  __u16 id;
  __u16 frag_off;
  __u8 ttl;
  __u8 protocol;
  __u16 check;
  __u32 saddr;
  __u32 daddr;
} __attribute__((packed));

// TCP header
struct tcphdr {
  __u16 source;
  __u16 dest;
  __u32 seq;
  __u32 ack_seq;
  union {
    struct {
      // Field order has been converted LittleEndiand -> BigEndian
      // in order to simplify flag checking (no need to ntohs())
      __u16 ns : 1,
      reserved : 3,
      doff : 4,
      fin : 1,
      syn : 1,
      rst : 1,
      psh : 1,
      ack : 1,
      urg : 1,
      ece : 1,
      cwr : 1;
    };
  };
  __u16 window;
  __u16 check;
  __u16 urg_ptr;
};

SEC("xdp")
int rewrite_ip(struct xdp_md *ctx)
{
  void* data_end = (void*)(long)ctx->data_end;
  void* data = (void*)(long)ctx->data;

  // L2
  struct ethhdr *ether = data;
  if (data + sizeof(*ether) > data_end) {
    return XDP_ABORTED;
  }

  // L3
  if (ether->h_proto != 0x08) { // htons(ETH_P_IP) -> 0x08
    // Non IPv4
    return XDP_PASS;
  }
  data += sizeof(*ether);
  struct iphdr *ip = data;
  if (data + sizeof(*ip) > data_end) {
    return XDP_ABORTED;
  }

  // L4
  if (ip->protocol != 0x06) {  // IPPROTO_TCP -> 6
    // Non TCP
    return XDP_PASS;
  }
  data += ip->ihl * 4;
  struct tcphdr *tcp = data;
  if (data + sizeof(*tcp) > data_end) {
    return XDP_ABORTED;
  }


  if (tcphdr->dest == SWAP_ORDER_16(TARGET_PORT)) {
    // rewrite ip
    tcphdr->daddr = htonl(REWRITE_IP);
    // update checksum
  }

  return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
