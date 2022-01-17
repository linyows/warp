#include "bpf_helpers.h"

#define TARGET_PORT 25
#define OVERRIDE_IP "192.168.30.30"

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
  // read data
  void* data_end = (void*)(long)ctx->data_end;
  void* data = (void*)(long)ctx->data;

  struct ethhdr *ether = data;
  // L2: frame header size
  if (data + sizeof(*ether) > data_end) {
    return XDP_ABORTED;
  }

  // L3: non ipv4?
  //if (ether->h_proto != htons(ETH_P_IP)) {
  if (ether->h_proto != 0x08) {
    return XDP_PASS;
  }
  data += sizeof(*ether);
  struct iphdr *ip = data;
  // ip header size
  if (data + sizeof(*ip) > data_end) {
    return XDP_ABORTED;
  }

  // L4: non tcp?
  //if (ip->protocol != IPPROTO_TCP) {
  if (ip->protocol != 0x06) {
    return XDP_PASS;
  }
  data += ip->ihl * 4;
  struct tcphdr *tcp = data;
  // tcp header size
  if (data + sizeof(*tcp) > data_end) {
    return XDP_ABORTED;
  }

  // target ip?
  unsigned long tip = htonl(inet_addr(OVERRIDE_IP)
  if (ip->daddr == tip || ip->saddr == tip) {
    return XDP_PASS;
  }

  // non target port?
  if (tcphdr->dest != htons(TARGET_PORT)) {
    return XDP_PASS;
  }

  // override ip header
  unsigned short old_daddr;
  old_daddr = ntohs(*(unsigned short *)&ip->daddr);
  ip->tos = 7 << 2;
  ip->daddr = htonl(inet_addr(OVERRIDE_IP));
  ip->check = 0;
  ip->check = checksum((unsigned short *)ip, sizeof(struct iphdr));

  // update tcp checksum
  unsigned long sum;
  sum = old_daddr + (~ntohs(*(unsigned short *)&ip->daddr) & 0xffff);
  sum += ntohs(tcphdr->check);
  sum = (sum & 0xffff) + (sum>>16);
  tcphdr->check = htons(sum + (sum>>16) + 1);

  return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
