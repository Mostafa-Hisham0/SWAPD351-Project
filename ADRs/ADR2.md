# ADR 2: Use RabbitMQ as the Message Broker
**context**
A message broker is like a post office for messages. 
When you send a message in a chat app, it doesn't go directly to the other person like : when the system is busy ,something goes wrong or the recipient is offline
it goes to the message broker first then the broker  delivers the message to the recipient
that ensures messages are delivered reliably.
# Reason it was chosen:
 It uses queues, retries, and acknowledgments to make sure no message is lost
 in case  a message can’t be delivered after multiple retries it gets moved to dead-letter queue and this ensures that no message is ever completely lost, and you can check the queue to see what went wrong
 not only does it prevent the messages of getting lost ,but it also balances the workload ,so it prevents the system from failing or getting overwhelmed.
 RabbitMQ is easy to set up and integrate with other technologies like WebSockets for real-time messaging.

# Alternatives Considered:
1)Kafka

Pros: 
Great for handling huge amounts of data.
Cons: More complex to set up and manage.

2)Redis:

Pros: 
Fast and lightweight.
Cons: Not as reliable as RabbitMQ for handling lots of messages.

# Decision taken:

we will use
RabbitMQ is the best choice for our chat system because of its reliability and scalability
where it uses message queuing and acknowledgments.