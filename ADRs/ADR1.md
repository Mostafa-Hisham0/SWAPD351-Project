
# ADR 1: Use OAuth2 with Google for Authentication

**context**
Authentication is how users log in to the real-chat app we have two ways either use your own email
and password or just use your Google account.

# Reason it was chosen :
We use OAuth2 with Google for Authentication because this option is very important
as a lot of customers find it much easier to just use their Google account as it is much faster not only that ,but it is of 
advantage to us as we don't have to store the data in our database so (better security).

# Alternatives Considered:
No other alternatives

**Pros**: 
Easier to implement
No password needed 
credentials stored in google
user-friendly as most people use this way for authentication 
easy to Set up.

**cons**:
Harder to implement.
some people don't have Google accounts.

# Decision taken:
We will use Google OAuth2 for third-party authentication.
we will use email and password with secure password hashing for traditional authentication.

