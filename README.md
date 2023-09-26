# stats

The goal of this project is to collect various statistics in a Slack org and post them every month for people to see.

Metrics to track:

- member with the most likes received
- member with the most dislikes received

The payload in the POST request to `/slack/monthly-update` should take the form in which the value to the date key represents `<year>-<month>`. The following example corresponds to October 2023.

```json
{
    "date":"2023-10"
}
```
