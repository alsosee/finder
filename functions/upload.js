export async function onRequest(context) {
  try {
    switch (context.request.method) {
      case 'PUT':
        const key = context.request.headers.get("x-file-name");
        await context.env.MEDIA.put(key, context.request.body);
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
