export async function onRequest(context) {
  try {
    switch (context.request.method) {
      case 'POST':
        await context.env.MEDIA.put("temp", context.request.body);
        return new Response(`Put ${"temp"} successfully!`);

      default:
        return new Response(`${context.request.method} is not allowed.`, {
          status: 405,
          headers: {
            Allow: 'PUT',
          },
        });
    }
  } catch (err) {
    return new Response(err.stack || err, { status: 500 });
  }
}
