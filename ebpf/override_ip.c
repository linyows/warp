#include <linux/bpf.h>
#include <arpa/inet.h>
#include <bpf/bpf_helpers.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
//#include <linux/ipv6.h>
#include <linux/tcp.h>
//#include <linux/if_packet.h>
//#include <linux/if_vlan.h>

#define TARGET_PORT 25
#define OVERRIDE_IP 505325760 // 192.168.30.30

static inline unsigned short checksum(unsigned short *buf, int bufsz)
{
  unsigned long sum = 0;

  while (bufsz > 1) {
    sum += *buf;
    buf++;
    bufsz -= 2;
  }

  if (bufsz == 1) {
    sum += *(unsigned char *)buf;
  }

  sum = (sum & 0xffff) + (sum >> 16);
  sum = (sum & 0xffff) + (sum >> 16);

  return ~sum;
}

/*
 * Ethernet header
 * https://github.com/torvalds/linux/blob/master/include/uapi/linux/if_ether.h#L169-L174
 * IPv4 header
 * https://github.com/torvalds/linux/blob/master/include/uapi/linux/ip.h#L86-L106
 * TCP header
 * https://github.com/torvalds/linux/blob/master/include/uapi/linux/tcp.h#L25-L58
 */

SEC("override_ip")
int override_ip_func(struct xdp_md *ctx)
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
  if (ether->h_proto != htons(ETH_P_IP)) {
    return XDP_PASS;
  }
  data += sizeof(*ether);
  struct iphdr *ip = data;
  // ip header size
  if (data + sizeof(*ip) > data_end) {
    return XDP_ABORTED;
  }

  // L4: non tcp?
  if (ip->protocol != IPPROTO_TCP) {
    return XDP_PASS;
  }
  data += ip->ihl * 4;
  struct tcphdr *tcp = data;
  // tcp header size
  if (data + sizeof(*tcp) > data_end) {
    return XDP_ABORTED;
  }

  // target ip?
  unsigned long tip = htonl(OVERRIDE_IP);
  if (ip->daddr == tip || ip->saddr == tip) {
    return XDP_PASS;
  }

  // non target port?
  if (tcp->dest != htons(TARGET_PORT)) {
    return XDP_PASS;
  }

  // override ip header
  unsigned short old_daddr;
  old_daddr = ntohs(*(unsigned short *)&ip->daddr);
  ip->tos = 7 << 2;
  ip->daddr = htonl(OVERRIDE_IP);
  ip->check = 0;
  ip->check = checksum((unsigned short *)ip, sizeof(struct iphdr));

  // update tcp checksum
  unsigned long sum;
  sum = old_daddr + (~ntohs(*(unsigned short *)&ip->daddr) & 0xffff);
  sum += ntohs(tcp->check);
  sum = (sum & 0xffff) + (sum>>16);
  tcp->check = htons(sum + (sum>>16) + 1);

  return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
