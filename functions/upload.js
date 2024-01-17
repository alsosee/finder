export async function onRequest(context) {
  try {
    switch (context.request.method) {
      case 'PUT':
        const key = context.request.headers.get("x-file-name");

        await context.env.MEDIA.put(key, context.request.body);

        const response = await fetch("https://api.github.com/repos/alsosee/media/dispatches", {
          method: "POST",
          headers: {
            "Authorization": `Bearer ${context.env.GHP_TOKEN}`,
            "Accept": "application/vnd.github.everest-preview+json",
            "Content-Type": "application/json",
            "User-Agent": "alsosee/finder/1.0.0 (CloudFlare Pages Function)"
          },
          body: JSON.stringify({
            event_type: "pull",
            client_payload: {
              path: key,
            },
          }),
        });

        if (response.status !== 204) {
          const text = await response.text();
          console.log(response.headers);
          console.log(text);
          throw new Error(`GitHub API failed with status code ${response.status}`);
        }

        return new Response(
          JSON.stringify({ status: "ok", key: key }),
          { status: 200 }
        );

      default:
        return new Response(
          JSON.stringify({
            error: {
              message: 'Method Not Allowed',
            }
          }),
          {
            status: 405,
            headers: {
              Allow: 'PUT',
            },
          }
        );
    }
  } catch (err) {
    return new Response(
      JSON.stringify({ error: err.stack || err }),
      { status: 500 }
    );
  }
}
