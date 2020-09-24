# Foodtruck

## Required Environment Variables

`AZURE_TENANT_ID`
`AZURE_CLIENT_ID`
`AZURE_CLIENT_SECRET`
`AZURE_SUBSCRIPTION_ID`
`EVENTHUB_NAMESPACE`

## azeventhub

### azure-event-hubs

**IMPORTANT**
An assumption is made prior to running this code that an Eventhub namespace is deployed in Azure and the proper environment variables are set.

#### RegisterNode

- Register node will create an event hub in the EVENTHUB_NAMESPACE (this should be set as an env var)

- The event hub name will be the hostname of the machine

#### ListenToHub

- A service principal must have proper explicit permissions to listen on the event hub namespace.  [Microsoft Explains it here](https://docs.microsoft.com/en-us/azure/event-hubs/authorize-access-azure-active-directory)

- Gets a token for auth using the environment variables set for the service principal

- Listens to the hub created from RegisterNode

